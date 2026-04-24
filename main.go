package main

import (
	"kerbecs/admin"
	"kerbecs/config"
	"kerbecs/gateway"
	"kerbecs/provider"
	"kerbecs/router"
	"kerbecs/utils"

	"golang.org/x/sync/errgroup"
)

var eg errgroup.Group

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

	eg.Go(func() error {
		return admin.StartServer()
	})
	eg.Go(func() error {
		return gateway.StartServer(rt)
	})
	if err := eg.Wait(); err != nil {
		utils.SugarLogger.Fatalf("Failed to start servers: %v", err)
	}
}
