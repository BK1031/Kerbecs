package main

import (
	"kerbecs/config"
	"kerbecs/controller"
	"kerbecs/service"
	"kerbecs/utils"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	config.PrintStartupBanner()
	utils.InitializeLogger()
	defer utils.Logger.Sync()

	utils.VerifyConfig()
	service.RegisterRincon()

	adminRouter := controller.SetupRouter()
	controller.InitializeAdminRoutes(adminRouter)
	go func() {
		err := adminRouter.Run(":" + config.AdminPort)
		if err != nil {
			utils.SugarLogger.Fatalf("Failed to start admin gateway: %v", err)
		}
	}()

	// initialize a reverse proxy and pass the actual backend server url here
	proxy, err := NewProxy("http://localhost:7001")
	if err != nil {
		panic(err)
	}
	// handle all requests to your server using the proxy
	http.HandleFunc("/", ProxyRequestHandler(proxy))
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

// NewProxy takes target host and creates a reverse proxy
func NewProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}

	return httputil.NewSingleHostReverseProxy(url), nil
}

// ProxyRequestHandler handles the http request using proxy
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}
