package gateway

import (
	"kerbecs/config"
	"kerbecs/router"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func StartServer(cfg HandlerConfig, rt *router.Router) error {
	engine := SetupRouter(cfg, rt)
	return engine.Run(":" + config.Port)
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
