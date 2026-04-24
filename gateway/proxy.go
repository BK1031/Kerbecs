package gateway

import (
	"bytes"
	"io"
	"kerbecs/router"
	"kerbecs/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
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

// NewProxyHandler returns a gin handler that resolves routes via the given
// router and proxies matched requests to the route's upstream.
func NewProxyHandler(rt *router.Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		match := rt.Find(c.Request.Method, c.Request.Host, c.Request.URL.Path)
		if match == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No route configured for " + c.Request.Method + " " + c.Request.URL.String(),
			})
			return
		}

		up := match.Route.Upstream
		if len(up.Instances) == 0 {
			c.JSON(http.StatusBadGateway, gin.H{
				"message": "No instances available for upstream " + up.Name,
			})
			return
		}

		target, err := url.Parse(up.Instances[0])
		if err != nil {
			utils.SugarLogger.Errorf("upstream %q: invalid endpoint %q: %v", up.Name, up.Instances[0], err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Invalid upstream endpoint for " + up.Name,
			})
			return
		}

		if match.Route.Rewrite != nil {
			newPath := router.RewritePath(c.Request.URL.Path, match.Route.Rewrite)
			c.Request.URL.Path = newPath
			c.Request.URL.RawPath = ""
		}

		utils.SugarLogger.Infof("PROXY TO: %s @ %s%s", up.FormattedNameWithVersion(), target.String(), c.Request.URL.Path)

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			utils.SugarLogger.Errorf("failed to reach upstream %q: %v", up.Name, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message":"upstream unreachable: ` + up.Name + `"}`))
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
