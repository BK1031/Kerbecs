package main

import (
	"kerbecs/config"
	"kerbecs/service"
	"kerbecs/utils"
)

func main() {
	config.PrintStartupBanner()
	utils.InitializeLogger()
	defer utils.Logger.Sync()

	service.RegisterRincon()

}
