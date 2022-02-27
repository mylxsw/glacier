package log

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"sync"
)

var defaultLogger infra.Logger = log.Module("glacier")
var lock sync.RWMutex

func SetDefaultLogger(logger infra.Logger) {
	lock.Lock()
	defer lock.Unlock()

	defaultLogger = logger
}

func Default() infra.Logger {
	lock.RLock()
	defer lock.RUnlock()

	return defaultLogger
}

func Debug(v ...interface{}) {
	Default().Debug(v...)
}

func Debugf(format string, v ...interface{}) {
	Default().Debugf(format, v...)
}

func Info(v ...interface{}) {
	Default().Info(v...)
}

func Infof(format string, v ...interface{}) {
	Default().Infof(format, v...)
}

func Error(v ...interface{}) {
	Default().Error(v...)
}

func Errorf(format string, v ...interface{}) {
	Default().Errorf(format, v...)
}

func Warning(v ...interface{}) {
	Default().Warning(v...)
}

func Warningf(format string, v ...interface{}) {
	Default().Warningf(format, v...)
}

func Critical(v ...interface{}) {
	Default().Critical(v...)
}

func Criticalf(format string, v ...interface{}) {
	Default().Criticalf(format, v...)
}
