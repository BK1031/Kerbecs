package config

import (
	"os"

	"github.com/bk1031/rincon-go/v2"
)

var Service rincon.Service = rincon.Service{
	Name:    "Kerbecs",
	Version: "1.3.0",
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

var KerbecsUser = os.Getenv("KERBECS_USER")
var KerbecsPassword = os.Getenv("KERBECS_PASSWORD")
var KerbecsEndpoint = os.Getenv("KERBECS_ENDPOINT")
var KerbecsHealthCheck = os.Getenv("KERBECS_HEALTH_CHECK")

var RinconClient *rincon.Client
var RinconUser = os.Getenv("RINCON_USER")
var RinconPassword = os.Getenv("RINCON_PASSWORD")
var RinconEndpoint = os.Getenv("RINCON_ENDPOINT")
