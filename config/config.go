package config

import (
	"github.com/bk1031/rincon-go"
	"os"
)

var Version = "1.0.0"
var Env = os.Getenv("ENV")
var Port = os.Getenv("PORT")
var AdminPort = os.Getenv("ADMIN_PORT")

var AuthUser = os.Getenv("AUTH_USER")
var AuthPassword = os.Getenv("AUTH_PASSWORD")

var RinconClient *rincon.Client
var RinconUser = os.Getenv("RINCON_USER")
var RinconPassword = os.Getenv("RINCON_PASSWORD")
