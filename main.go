package main

import (
	"kerbecs/config"
	"kerbecs/controller"
	"kerbecs/service"
	"kerbecs/utils"

	"golang.org/x/sync/errgroup"
)

var eg errgroup.Group

func main() {
	config.PrintStartupBanner()
	utils.InitializeLogger()
	defer utils.Logger.Sync()

	utils.VerifyConfig()
	service.RegisterRincon()
	eg.Go(func() error {
		return controller.StartAdminServer()
	})
	eg.Go(func() error {
		return controller.StartProxyServer()
	})
	if err := eg.Wait(); err != nil {
		utils.SugarLogger.Fatalf("Failed to start servers: %v", err)
	}
}
