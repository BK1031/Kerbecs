package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"kerbecs/config"
	"kerbecs/model"
	"kerbecs/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
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
		duration := time.Since(startTime.(time.Time)).Milliseconds()

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
	startTime, _ := c.Get("Request-Start-Time")
	service, err := config.RinconClient.MatchRoute(c.Request.URL.Path, c.Request.Method)
	if err != nil {
		c.JSON(404, model.Response{
			Status:    "ERROR",
			Ping:      strconv.FormatInt(time.Since(startTime.(time.Time)).Milliseconds(), 10) + "ms",
			Gateway:   config.Service.FormattedNameWithVersion(),
			Service:   config.RinconClient.Rincon().FormattedNameWithVersion(),
			Timestamp: time.Now().Format("Mon Jan 02 15:04:05 MST 2006"),
			Data:      json.RawMessage("{\"message\": \"No service to handle route: " + c.Request.URL.String() + "\"}"),
		})
		return
	}
	utils.SugarLogger.Infoln("PROXY TO: (" + strconv.Itoa(service.ID) + ") " + service.Name + " @ " + service.Endpoint)
	endpoint, err := url.Parse(service.Endpoint)
	if err != nil {
		c.JSON(500, model.Response{
			Status:    "ERROR",
			Ping:      strconv.FormatInt(time.Since(startTime.(time.Time)).Milliseconds(), 10) + "ms",
			Gateway:   config.Service.FormattedNameWithVersion(),
			Service:   config.RinconClient.Rincon().FormattedNameWithVersion(),
			Timestamp: time.Now().Format("Mon Jan 02 15:04:05 MST 2006"),
			Data:      json.RawMessage("{\"message\": \"Failed to parse service endpoint: " + service.Endpoint + "\"}"),
		})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(endpoint)
	proxy.ModifyResponse = func(response *http.Response) error {
		if response.StatusCode == http.StatusSwitchingProtocols {
			return nil // it's a WebSocket upgrade, leave it untouched!
		}
		respModel, err := BuildResponseStruct(response, *service)
		if err != nil {
			return err
		}
		respModel.Timestamp = time.Now().Format("Mon Jan 02 15:04:05 MST 2006")
		respModel.Ping = strconv.FormatInt(time.Since(startTime.(time.Time)).Milliseconds(), 10) + "ms"
		b, _ := json.Marshal(respModel)
		response.Body = io.NopCloser(bytes.NewReader(b))
		response.ContentLength = int64(len(b))
		response.Header.Set("Content-Length", strconv.Itoa(len(b)))
		return nil
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		utils.SugarLogger.Errorln("Failed to proxy request: " + err.Error())
		writer.WriteHeader(http.StatusBadGateway)
		respModel := model.Response{
			Status:    "ERROR",
			Ping:      strconv.FormatInt(time.Since(startTime.(time.Time)).Milliseconds(), 10) + "ms",
			Gateway:   config.Service.FormattedNameWithVersion(),
			Service:   service.FormattedNameWithVersion(),
			Timestamp: time.Now().Format("Mon Jan 02 15:04:05 MST 2006"),
		}
		respModel.Data = json.RawMessage("{\"message\": \"Failed to reach " + service.Name + ": " + err.Error() + "\"}")
		b, _ := json.Marshal(respModel)
		writer.Write(b)
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}
