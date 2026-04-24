package admin

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"kerbecs/config"
	"kerbecs/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 30 * time.Second

// Serve starts the admin HTTP listener and blocks until ctx is canceled, at
// which point it drains in-flight requests up to shutdownTimeout.
func Serve(ctx context.Context) error {
	engine := SetupRouter()
	InitializeRoutes(engine)
	srv := &http.Server{
		Addr:    ":" + config.AdminPort,
		Handler: engine,
	}

	errCh := make(chan error, 1)
	go func() {
		utils.SugarLogger.Infof("admin listening on :%s", config.AdminPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		utils.SugarLogger.Infoln("admin: draining")
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

func SetupRouter() *gin.Engine {
	if config.Env == "PROD" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: true,
	}))
	r.Use(AuthMiddleware())
	return r
}

func InitializeRoutes(router *gin.Engine) {
	gw := router.Group("/admin-gw", func(c *gin.Context) {})
	gw.GET("/ping", Ping)
}

func AuthMiddleware() gin.HandlerFunc {
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
			if len(pair) != 2 || !constantTimeEqual(pair[0], config.KerbecsUser) || !constantTimeEqual(pair[1], config.KerbecsPassword) {
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
