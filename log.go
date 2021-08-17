package ble

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Info(...interface{})
	Debug(...interface{})
	Error(...interface{})
	Warn(...interface{})

	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Errorf(string, ...interface{})
	Warnf(string, ...interface{})

	ChildLogger(tags map[string]interface{}) Logger
}

var logger Logger
var loggerMu sync.Mutex

func SetLogLevelMax() {
	l := GetLogger()

	if lg, ok := l.(*defaultLogger); ok {
		lg.Entry.Logger.SetLevel(logrus.TraceLevel)
	} else {
		// clean this up later
		l.Error("non-default logger, don't know how to set level")
	}
}

func SetLogger(l Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}

func GetLogger() Logger {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if logger == nil {
		logger = buildDefaultLogger()
	}

	return logger
}

type defaultLogger struct {
	*logrus.Entry
}

func buildDefaultLogger() Logger {
	l := &logrus.Logger{
		Formatter: &logrus.TextFormatter{DisableTimestamp: true},
		Level:     logrus.InfoLevel,
		Out:       os.Stderr,
		Hooks:     make(logrus.LevelHooks),
	}

	return &defaultLogger{Entry: l.WithFields(map[string]interface{}{})}
}

func (d *defaultLogger) ChildLogger(ff map[string]interface{}) Logger {
	nl := &defaultLogger{d.Entry.WithFields(ff)}
	return nl
}
