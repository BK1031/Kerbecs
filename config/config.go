package config

import (
	"os"

	"github.com/bk1031/rincon-go/v2"
)

var Service rincon.Service = rincon.Service{
	Name:    "Kerbecs",
	Version: "1.2.0",
}

var Routes = []rincon.Route{
	{
		Route:  "/admin-gw/**",
		Method: "*",
	},
}

var Env = os.Getenv("ENV")
var Port = os.Getenv("PORT")
var AdminPort = os.Getenv("ADMIN_PORT")

var AdminUser = os.Getenv("ADMIN_USER")
var AdminPassword = os.Getenv("ADMIN_PASSWORD")

var RinconClient *rincon.Client
var RinconUser = os.Getenv("RINCON_USER")
var RinconPassword = os.Getenv("RINCON_PASSWORD")
