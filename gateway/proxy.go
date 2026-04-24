package gateway

import (
	"bytes"
	"io"
	"kerbecs/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
		if strings.ToLower(c.GetHeader("Upgrade")) != "" {
			utils.SugarLogger.Infoln("UPGRADE: " + c.GetHeader("Upgrade"))
		}
		c.Next()
	}
}

func ProxyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func ProxyResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime, _ := c.Get("Request-Start-Time")
		c.Next()

		var duration int64
		if t, ok := startTime.(time.Time); ok {
			duration = time.Since(t).Milliseconds()
		}

		status := c.Writer.Status()
		isWebSocket := strings.ToLower(c.GetHeader("Upgrade")) == "websocket"

		if isWebSocket {
			utils.SugarLogger.Infof("WS STATUS: %d – took %dms", status, duration)
		} else {
			utils.SugarLogger.Infof("RESPONSE STATUS: %d – took %dms", status, duration)
		}
	}
}

func ProxyHandler(c *gin.Context) {
	c.JSON(404, gin.H{
		"message": "No route configured for " + c.Request.Method + " " + c.Request.URL.String(),
	})
}
