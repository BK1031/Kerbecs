package controller

import (
	"bytes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"io"
	"kerbecs/config"
	"kerbecs/model"
	"kerbecs/utils"
	"strconv"
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
	r.Use(ProxyRequestLogger())
	r.Use(ProxyAuthMiddleware())
	r.Use(ProxyResponseLogger())
	return r
}

func InitializeProxyRoutes(router *gin.Engine) {
	router.Any("/*path", ProxyHandler)
}

func ProxyRequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, _ := uuid.NewV7()
		c.Set("Request-ID", requestID.String())
		c.Set("Request-Start-Time", time.Now())
		utils.SugarLogger.Infoln("-------------------------------------------------------------------")
		utils.SugarLogger.Infoln(time.Now().Format("Mon Jan 02 15:04:05 MST 2006"))
		utils.SugarLogger.Infoln("REQUEST ID: " + requestID.String())
		utils.SugarLogger.Infoln("REQUEST ROUTE: " + c.Request.Host + c.Request.URL.String() + " [" + c.Request.Method + "]")
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			utils.SugarLogger.Infoln("REQUEST BODY: " + err.Error())
		} else {
			utils.SugarLogger.Infoln("REQUEST BODY: " + string(bodyBytes))
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		utils.SugarLogger.Infoln("REQUEST ORIGIN: " + c.ClientIP())
		c.Request.Header.Set("Request-ID", requestID.String())
		c.Next()
	}
}

func ProxyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func ProxyHandler(c *gin.Context) {
	//requestID := c.GetHeader("Request-ID")
	startTime, _ := c.Get("Request-Start-Time")
	println(c.Request.URL.Path)
	service, err := config.RinconClient.MatchRoute(c.Request.URL.Path)
	if err != nil {
		c.JSON(404, model.Response{
			Status:    "ERROR",
			Ping:      strconv.FormatInt(time.Now().Sub(startTime.(time.Time)).Milliseconds(), 10) + "ms",
			Gateway:   "Kerbecs v" + config.Version,
			Service:   "Rincon",
			Timestamp: time.Now().Format("Mon Jan 02 15:04:05 MST 2006"),
			Data:      json.RawMessage("{\"message\": \"No service to handle route: " + c.Request.URL.String() + "\"}"),
		})
		return
	}
	println(service.Name)
	c.JSON(200, gin.H{"message": "Proxying request"})
}

func ProxyResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		utils.SugarLogger.Infoln("RESPONSE STATUS: " + strconv.Itoa(c.Writer.Status()))
	}
}
