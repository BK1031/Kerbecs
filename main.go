package main

import (
	"context"
	"kerbecs/admin"
	"kerbecs/config"
	"kerbecs/gateway"
	"kerbecs/pkg/logger"
	"kerbecs/pkg/middleware"
	"kerbecs/provider"
	"kerbecs/router"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

func main() {
	logger.Init(os.Getenv("ENV") == "PROD")
	defer logger.Logger.Sync()

	middleware.RegisterBuiltins()

	path := config.FilePath()
	file, err := config.LoadFile(path)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to load config %s: %v", path, err)
	}
	for _, w := range config.ApplyDefaults(file) {
		logger.SugarLogger.Warnln(w)
	}
	logger.SugarLogger.Infof("Loaded config from %s", path)

	config.PrintStartupBanner(file.Gateway.Env)

	mwRegistry, err := middleware.BuildRegistry(file.Middlewares)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to build middleware registry: %v", err)
	}

	static, err := provider.NewStatic(file, mwRegistry)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to build static provider: %v", err)
	}
	logger.SugarLogger.Infof("Static provider loaded %d route(s)", len(static.Routes()))

	rt, err := router.New(static)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to build router: %v", err)
	}

	gatewayChain, err := mwRegistry.Chain(file.Listeners.Gateway.Middlewares)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to build gateway listener middleware chain: %v", err)
	}

	handlerCfg := gateway.HandlerConfig{
		GatewayName:    firstNonEmpty(file.Gateway.Name, config.Name),
		GatewayVersion: firstNonEmpty(file.Gateway.Version, config.Version),
	}
	listenerCfg := gateway.ListenerConfig{
		Port:        file.Listeners.Gateway.Port,
		Env:         file.Gateway.Env,
		CORS:        file.Listeners.Gateway.CORS,
		Middlewares: gatewayChain,
	}
	adminCfg := admin.Config{
		Port:     file.Listeners.Admin.Port,
		Env:      file.Gateway.Env,
		Username: file.Listeners.Admin.Auth.Username,
		Password: file.Listeners.Admin.Auth.Password,
		CORS:     file.Listeners.Admin.CORS,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var eg errgroup.Group
	eg.Go(func() error { return admin.Serve(ctx, adminCfg) })
	eg.Go(func() error { return gateway.Serve(ctx, listenerCfg, handlerCfg, rt) })

	if err := eg.Wait(); err != nil {
		logger.SugarLogger.Fatalf("server error: %v", err)
	}
	logger.SugarLogger.Infoln("shutdown complete")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
