package controller

import (
	"encoding/base64"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"kerbecs/config"
	"strings"
	"time"
)

func StartProxyServer() error {
	proxyRouter := SetupProxyRouter()
	InitializeProxyRoutes(proxyRouter)
	return proxyRouter.Run(":" + config.Port)
}

func SetupProxyRouter() *gin.Engine {
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
	r.Use(ProxyAuthMiddleware())
	return r
}

func InitializeProxyRoutes(router *gin.Engine) {
	router.Any("/*path", ProxyHandler)
}

func ProxyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() != "/admin-gw/ping" {
			auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
			if len(auth) != 2 || auth[0] != "Basic" {
				c.AbortWithStatusJSON(401, gin.H{"message": "Request not authorized"})
				return
			}
			payload, _ := base64.StdEncoding.DecodeString(auth[1])
			pair := strings.SplitN(string(payload), ":", 2)
			if len(pair) != 2 || pair[0] != config.AdminUser || pair[1] != config.AdminPassword {
				c.AbortWithStatusJSON(401, gin.H{"message": "Invalid credentials"})
				return
			}
		}
		c.Next()
	}
}

func ProxyHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Proxying request"})
}
