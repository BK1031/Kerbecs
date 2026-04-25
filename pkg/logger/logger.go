package logger

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger
var SugarLogger *zap.SugaredLogger

// Init sets up the zap logger. Passing production=true uses the JSON
// production config; false uses the development (pretty) config with caller
// info and error-level stacktraces.
func Init(production bool) {
	Logger = zap.Must(zap.NewProduction())
	if !production {
		Logger = zap.Must(zap.NewDevelopment(zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)))
	}
	SugarLogger = Logger.Sugar()
}
