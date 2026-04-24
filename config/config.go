package config

import "os"

const (
	Name    = "Kerbecs"
	Version = "2.2.0"
)

func FormattedNameWithVersion() string {
	return Name + ":v" + Version
}

var Env = os.Getenv("ENV")
var Port = os.Getenv("PORT")
var AdminPort = os.Getenv("ADMIN_PORT")

var KerbecsUser = os.Getenv("KERBECS_USER")
var KerbecsPassword = os.Getenv("KERBECS_PASSWORD")

var UseCors = os.Getenv("USE_CORS")
