package main

import (
	"context"
	"kerbecs/admin"
	"kerbecs/config"
	"kerbecs/gateway"
	"kerbecs/provider"
	"kerbecs/router"
	"kerbecs/utils"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

func main() {
	config.PrintStartupBanner()
	utils.InitializeLogger()
	defer utils.Logger.Sync()

	utils.VerifyConfig()

	path := config.FilePath()
	file, err := config.LoadFile(path)
	if err != nil {
		utils.SugarLogger.Fatalf("Failed to load config %s: %v", path, err)
	}
	utils.SugarLogger.Infof("Loaded config from %s", path)

	static, err := provider.NewStatic(file)
	if err != nil {
		utils.SugarLogger.Fatalf("Failed to build static provider: %v", err)
	}
	utils.SugarLogger.Infof("Static provider loaded %d route(s)", len(static.Routes()))

	rt, err := router.New(static)
	if err != nil {
		utils.SugarLogger.Fatalf("Failed to build router: %v", err)
	}

	handlerCfg := gateway.HandlerConfig{
		GatewayName:    firstNonEmpty(file.Gateway.Name, config.Name),
		GatewayVersion: firstNonEmpty(file.Gateway.Version, config.Version),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var eg errgroup.Group
	eg.Go(func() error { return admin.Serve(ctx) })
	eg.Go(func() error { return gateway.Serve(ctx, handlerCfg, rt) })

	if err := eg.Wait(); err != nil {
		utils.SugarLogger.Fatalf("server error: %v", err)
	}
	utils.SugarLogger.Infoln("shutdown complete")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
