package service

import (
	"github.com/bk1031/rincon-go"
	"kerbecs/config"
	"kerbecs/utils"
)

func RegisterRincon() {
	client, err := rincon.NewClient(rincon.Config{
		BaseURL:           "http://localhost:10311",
		HeartbeatMode:     rincon.ServerHeartbeat,
		HeartbeatInterval: 60,
		AuthUser:          config.RinconUser,
		AuthPassword:      config.RinconPassword,
	})
	if err != nil {
		utils.SugarLogger.Errorf("Failed to create Rincon client: %v", err)
		return
	}
	config.RinconClient = client
	id, err := config.RinconClient.Register(rincon.Service{
		Name:        "kerbecs",
		Version:     config.Version,
		Endpoint:    "http://localhost:" + config.AdminPort,
		HealthCheck: "http://localhost:" + config.AdminPort + "/ping",
	}, []string{"/admin-gw/**"})
	if err != nil {
		utils.SugarLogger.Errorf("Failed to register service with Rincon: %v", err)
		return
	}
	utils.SugarLogger.Infof("Registered service with ID: %d", id)
}
