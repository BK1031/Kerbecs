package gateway

import (
	"kerbecs/config"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func StartServer() error {
	router := SetupRouter()
	InitializeRoutes(router)
	return router.Run(":" + config.Port)
}

func SetupRouter() *gin.Engine {
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
	return r
}

func InitializeRoutes(router *gin.Engine) {
	router.Any("/*path", ProxyHandler)
}
