package gateway

import (
	"context"
	"errors"
	"fmt"
	"kerbecs/config"
	"kerbecs/router"
	"kerbecs/utils"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 30 * time.Second

// Serve starts the gateway HTTP listener and blocks until ctx is canceled, at
// which point it drains in-flight requests up to shutdownTimeout.
func Serve(ctx context.Context, cfg HandlerConfig, rt *router.Router) error {
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: SetupRouter(cfg, rt),
	}

	errCh := make(chan error, 1)
	go func() {
		utils.SugarLogger.Infof("gateway listening on :%s", config.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		utils.SugarLogger.Infoln("gateway: draining")
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

func SetupRouter(cfg HandlerConfig, rt *router.Router) *gin.Engine {
	if config.Env == "PROD" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	if config.UseCors == "true" {
		r.Use(cors.New(cors.Config{
			AllowAllOrigins:  true,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
			MaxAge:           12 * time.Hour,
			AllowCredentials: true,
		}))
	}
	r.Use(ProxyRequestLogger())
	r.Use(ProxyAuthMiddleware())
	r.Use(ProxyResponseLogger())
	r.Any("/*path", NewProxyHandler(cfg, rt))
	return r
}
