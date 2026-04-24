package utils

import (
	"os"

	"go.uber.org/zap"
)

var Logger *zap.Logger
var SugarLogger *zap.SugaredLogger

// InitializeLogger sets up the zap logger. Before the config file has been
// loaded we bootstrap off the ENV environment variable: DEV gives the
// development logger (pretty output), anything else gives the production
// logger (structured JSON).
func InitializeLogger() {
	Logger = zap.Must(zap.NewProduction())
	if os.Getenv("ENV") == "DEV" {
		Logger = zap.Must(zap.NewDevelopment())
	}
	SugarLogger = Logger.Sugar()
}
