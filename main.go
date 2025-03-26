package main

import (
	"kerbecs/admin"
	"kerbecs/config"
	"kerbecs/gateway"
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
		return admin.StartServer()
	})
	eg.Go(func() error {
		return gateway.StartServer()
	})
	if err := eg.Wait(); err != nil {
		utils.SugarLogger.Fatalf("Failed to start servers: %v", err)
	}
}
