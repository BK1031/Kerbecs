package admin

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"kerbecs/config"
	"kerbecs/pkg/logger"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 30 * time.Second

// Config is everything the admin listener needs from the loaded config file.
type Config struct {
	Port     string
	Env      string
	Username string
	Password string
	CORS     *config.CORSConfig
}

// Serve starts the admin HTTP listener and blocks until ctx is canceled, at
// which point it drains in-flight requests up to shutdownTimeout. currentRouter
// returns the live router so the registry endpoints reflect the latest config
// after a hot reload.
func Serve(ctx context.Context, cfg Config, currentRouter RouterFunc) error {
	engine := SetupRouter(cfg)
	InitializeRoutes(engine, cfg, currentRouter)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: engine,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.SugarLogger.Infof("admin listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.SugarLogger.Infoln("admin: draining")
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("admin shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func SetupRouter(cfg Config) *gin.Engine {
	if cfg.Env == "PROD" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	if cfg.CORS != nil && cfg.CORS.Enabled {
		r.Use(cors.New(buildCORSConfig(cfg.CORS)))
	}
	r.Use(AuthMiddleware(cfg.Username, cfg.Password))
	return r
}

func InitializeRoutes(router *gin.Engine, cfg Config, currentRouter RouterFunc) {
	gw := router.Group("/admin-gw", func(c *gin.Context) {})
	gw.GET("/ping", Ping)
	gw.GET("/resolve", Resolve(currentRouter))
	gw.GET("/routes", Routes(currentRouter))
	gw.GET("/upstreams", Upstreams(currentRouter))
	gw.GET("/info", Info(cfg.Env, currentRouter))
}

func AuthMiddleware(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() != "/admin-gw/ping" {
			auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
			if len(auth) != 2 || auth[0] != "Basic" {
				c.AbortWithStatusJSON(401, gin.H{"message": "Request not authorized"})
				return
			}
			payload, err := base64.StdEncoding.DecodeString(auth[1])
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{"message": "Invalid credentials"})
				return
			}
			pair := strings.SplitN(string(payload), ":", 2)
			if len(pair) != 2 || !constantTimeEqual(pair[0], username) || !constantTimeEqual(pair[1], password) {
				c.AbortWithStatusJSON(401, gin.H{"message": "Invalid credentials"})
				return
			}
		}
		c.Next()
	}
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// buildCORSConfig translates a config.CORSConfig into a gin-contrib/cors
// config, filling in reasonable method/header defaults when unset.
func buildCORSConfig(c *config.CORSConfig) cors.Config {
	out := cors.Config{
		AllowAllOrigins:  c.AllowAllOrigins,
		AllowOrigins:     c.AllowedOrigins,
		AllowMethods:     c.AllowedMethods,
		AllowHeaders:     c.AllowedHeaders,
		AllowCredentials: c.AllowCredentials,
		MaxAge:           c.MaxAge.AsDuration(),
	}
	if len(out.AllowMethods) == 0 {
		out.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	}
	if len(out.AllowHeaders) == 0 {
		out.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	}
	if out.MaxAge == 0 {
		out.MaxAge = 12 * time.Hour
	}
	return out
}
