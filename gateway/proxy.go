package gateway

import (
	"encoding/json"
	"errors"
	"kerbecs/provider"
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

// HandlerConfig carries the gateway-level identity that feeds the envelope.
type HandlerConfig struct {
	GatewayName    string
	GatewayVersion string
}

func (h HandlerConfig) formattedGateway() string {
	return h.GatewayName + ":v" + h.GatewayVersion
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
// router and proxies matched requests to the route's upstream. If the matched
// route has envelope: default, the upstream response is buffered and wrapped
// in the Kerbecs envelope. passthrough routes stream unchanged.
//
// Pre-match errors (no route found) are returned as plain JSON without an
// envelope, since the envelope is a property of a matched route.
func NewProxyHandler(cfg HandlerConfig, rt *router.Router) gin.HandlerFunc {
	gateway := cfg.formattedGateway()
	transports := buildTransportCache(rt)

	return func(c *gin.Context) {
		start := requestStart(c)

		match := rt.Find(c.Request.Method, c.Request.Host, c.Request.URL.Path)
		if match == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No route configured for " + c.Request.Method + " " + c.Request.URL.String(),
			})
			return
		}

		up := match.Route.Upstream
		service := up.FormattedNameWithVersion()
		limits := match.Route.Limits

		// Request body cap — reject up front when Content-Length is known.
		if limits.MaxRequestBytes > 0 && c.Request.ContentLength > limits.MaxRequestBytes {
			writeError(c, match.Route.Envelope, http.StatusRequestEntityTooLarge, gateway, service, start,
				"request body exceeds max_request_bytes")
			return
		}
		// Wrap for streaming uploads where Content-Length is unknown or spoofed.
		if limits.MaxRequestBytes > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limits.MaxRequestBytes)
		}

		if len(up.Instances) == 0 {
			writeError(c, match.Route.Envelope, http.StatusBadGateway, gateway, service, start,
				"No instances available for upstream "+up.Name)
			return
		}

		target, err := url.Parse(up.Instances[0])
		if err != nil {
			utils.SugarLogger.Errorf("upstream %q: invalid endpoint %q: %v", up.Name, up.Instances[0], err)
			writeError(c, match.Route.Envelope, http.StatusInternalServerError, gateway, service, start,
				"Invalid upstream endpoint for "+up.Name)
			return
		}

		if match.Route.Rewrite != nil {
			c.Request.URL.Path = router.RewritePath(c.Request.URL.Path, match.Route.Rewrite)
			c.Request.URL.RawPath = ""
		}

		utils.SugarLogger.Infof("PROXY TO: %s @ %s%s", service, target.String(), c.Request.URL.Path)

		proxy := httputil.NewSingleHostReverseProxy(target)
		if tr, ok := transports[up.Name]; ok {
			proxy.Transport = tr
		}
		if match.Route.Envelope == provider.EnvelopeDefault {
			proxy.ModifyResponse = modifyResponseWithEnvelope(gateway, service, start, limits.MaxResponseBytes)
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			handleProxyError(w, match.Route.Envelope, gateway, service, start, up.Name, err)
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// handleProxyError translates a reverse-proxy error into a terminal response.
// Size-cap violations get their own status codes; everything else is 502.
func handleProxyError(w http.ResponseWriter, mode provider.EnvelopeMode, gateway, service string, start time.Time, upstream string, err error) {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		utils.SugarLogger.Warnf("request from client to %q exceeded max_request_bytes (limit %d)", upstream, maxBytes.Limit)
		writeRawError(w, mode, http.StatusRequestEntityTooLarge, gateway, service, start,
			"request body exceeds max_request_bytes")
		return
	}
	if errors.Is(err, errResponseTooLarge) {
		utils.SugarLogger.Warnf("upstream %q response exceeded max_response_bytes", upstream)
		writeRawError(w, mode, http.StatusBadGateway, gateway, service, start,
			"upstream response exceeds max_response_bytes")
		return
	}
	utils.SugarLogger.Errorf("failed to reach upstream %q: %v", upstream, err)
	writeRawError(w, mode, http.StatusBadGateway, gateway, service, start,
		"upstream unreachable: "+upstream+": "+err.Error())
}

// writeError emits an error response via gin.Context, enveloped if the route
// requested it.
func writeError(c *gin.Context, mode provider.EnvelopeMode, code int, gateway, service string, start time.Time, message string) {
	if mode == provider.EnvelopeDefault {
		body, err := envelopeFromMessage(code, gateway, service, start, message)
		if err != nil {
			c.JSON(code, gin.H{"message": message})
			return
		}
		c.Data(code, "application/json", body)
		return
	}
	c.JSON(code, gin.H{"message": message})
}

// writeRawError is the ResponseWriter equivalent of writeError, used from
// inside ReverseProxy hooks where gin.Context is no longer applicable.
func writeRawError(w http.ResponseWriter, mode provider.EnvelopeMode, code int, gateway, service string, start time.Time, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if mode == provider.EnvelopeDefault {
		body, err := envelopeFromMessage(code, gateway, service, start, message)
		if err == nil {
			_, _ = w.Write(body)
			return
		}
	}
	if fallback, err := json.Marshal(map[string]string{"message": message}); err == nil {
		_, _ = w.Write(fallback)
	} else {
		_, _ = w.Write([]byte(`{"message":"internal marshal error"}`))
	}
}

// requestStart returns the start time set by ProxyRequestLogger, falling back
// to now() if the middleware was not wired.
func requestStart(c *gin.Context) time.Time {
	if v, ok := c.Get("Request-Start-Time"); ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Now()
}
