package config

import (
	"github.com/bk1031/rincon-go"
	"os"
)

var Version = "1.0.0"
var Env = os.Getenv("ENV")
var Port = os.Getenv("PORT")

var AuthUser = os.Getenv("AUTH_USER")
var AuthPassword = os.Getenv("AUTH_PASSWORD")

var RinconClient rincon.Client
