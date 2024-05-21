package main

import (
	"kerbecs/config"
	"kerbecs/controller"
	"kerbecs/service"
	"kerbecs/utils"
)

func main() {
	config.PrintStartupBanner()
	utils.InitializeLogger()
	defer utils.Logger.Sync()

	utils.VerifyConfig()
	service.RegisterRincon()

	adminRouter := controller.SetupRouter()
	controller.InitializeAdminRoutes(adminRouter)
	err := adminRouter.Run(":" + config.AdminPort)
	if err != nil {
		utils.SugarLogger.Fatalf("Failed to start admin gateway: %v", err)
	}
}
