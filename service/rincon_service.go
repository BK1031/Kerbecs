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
	config.Service = rincon.Service{
		Name:        "Kerbecs",
		Version:     config.Version,
		Endpoint:    "http://localhost:" + config.AdminPort,
		HealthCheck: "http://host.docker.internal:" + config.AdminPort + "/admin-gw/ping",
	}
	id, err := config.RinconClient.Register(config.Service, []string{"/admin-gw/**"})
	if err != nil {
		utils.SugarLogger.Errorf("Failed to register service with Rincon: %v", err)
		return
	}
	config.Service = *config.RinconClient.Service()
	utils.SugarLogger.Infof("Registered service with ID: %d", id)
}
