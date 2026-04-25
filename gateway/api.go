package gateway

import (
	"context"
	"errors"
	"fmt"
	"kerbecs/config"
	"kerbecs/router"
	"kerbecs/pkg/logger"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 30 * time.Second

// ListenerConfig is everything the gateway listener needs from the loaded
// config file (transport concerns). Envelope identity flows through
// HandlerConfig.
type ListenerConfig struct {
	Port string
	Env  string
	CORS *config.CORSConfig
}

// Serve starts the gateway HTTP listener and blocks until ctx is canceled, at
// which point it drains in-flight requests up to shutdownTimeout.
func Serve(ctx context.Context, listener ListenerConfig, handler HandlerConfig, rt *router.Router) error {
	srv := &http.Server{
		Addr:    ":" + listener.Port,
		Handler: SetupRouter(listener, handler, rt),
	}

	errCh := make(chan error, 1)
	go func() {
		logger.SugarLogger.Infof("gateway listening on :%s", listener.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.SugarLogger.Infoln("gateway: draining")
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("gateway shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func SetupRouter(listener ListenerConfig, handler HandlerConfig, rt *router.Router) *gin.Engine {
	if listener.Env == "PROD" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	if listener.CORS != nil && listener.CORS.Enabled {
		r.Use(cors.New(buildCORSConfig(listener.CORS)))
	}
	r.Use(ProxyRequestLogger())
	r.Use(ProxyAuthMiddleware())
	r.Use(ProxyResponseLogger())
	r.Any("/*path", NewProxyHandler(handler, rt))
	return r
}

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
