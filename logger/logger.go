package logger

import (
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Init() {
	logger, err := zap.NewDevelopment() // Use Development logger for more verbose output
	if err != nil {
		panic("failed to initialize zap logger: " + err.Error())
	}
	Log = logger.Sugar()
}
