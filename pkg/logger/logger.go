package logger

import (
	"fmt"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"
)

func SetupLogger(level string) {
	loggerLevel, err := log.ParseLevel(level)
	log.SetReportCaller(true)

	log.SetFormatter(&log.JSONFormatter{
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return "", fmt.Sprintf("%s:%d", path.Base(frame.File), frame.Line)
		},
		TimestampFormat: "2006-01-02 15:04:05",
	})

	if err != nil {
		log.Infof("Level setup default INFO, err: %v", err)
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(loggerLevel)
	}
}
