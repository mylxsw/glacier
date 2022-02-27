package log

import (
	"fmt"
	"github.com/mylxsw/glacier/infra"
	"log"
	"os"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARNING
	ERROR
	CRITICAL
)

func hasLevel(c Level, levels []Level) bool {
	for _, l := range levels {
		if l == c {
			return true
		}
	}

	return false
}

func StdLogger(hideLevels ...Level) infra.Logger {
	return stdLogger{disallow: hideLevels}
}

type stdLogger struct {
	disallow []Level
}

func (s stdLogger) Debug(v ...interface{}) {
	if hasLevel(DEBUG, s.disallow) {
		return
	}

	log.Printf("[DEBUG] %s", fmt.Sprint(v...))
}

func (s stdLogger) Debugf(format string, v ...interface{}) {
	if hasLevel(DEBUG, s.disallow) {
		return
	}

	log.Printf("[DEBUG] %s", fmt.Sprintf(format, v...))
}

func (s stdLogger) Info(v ...interface{}) {
	if hasLevel(INFO, s.disallow) {
		return
	}

	log.Printf("[INFO] %s", fmt.Sprint(v...))
}

func (s stdLogger) Infof(format string, v ...interface{}) {
	if hasLevel(INFO, s.disallow) {
		return
	}

	log.Printf("[INFO] %s", fmt.Sprintf(format, v...))
}

func (s stdLogger) Error(v ...interface{}) {
	if hasLevel(ERROR, s.disallow) {
		return
	}

	log.Printf("[ERROR] %s", fmt.Sprint(v...))
}

func (s stdLogger) Errorf(format string, v ...interface{}) {
	if hasLevel(ERROR, s.disallow) {
		return
	}

	log.Printf("[ERROR] %s", fmt.Sprintf(format, v...))
}

func (s stdLogger) Warning(v ...interface{}) {
	if hasLevel(WARNING, s.disallow) {
		return
	}

	log.Printf("[WARNING] %s", fmt.Sprint(v...))
}

func (s stdLogger) Warningf(format string, v ...interface{}) {
	if hasLevel(WARNING, s.disallow) {
		return
	}

	log.Printf("[WARNING] %s", fmt.Sprintf(format, v...))
}

func (s stdLogger) Critical(v ...interface{}) {
	if !hasLevel(CRITICAL, s.disallow) {
		log.Printf("[CRITICAL] %s", fmt.Sprint(v...))
	}

	os.Exit(1)
}

func (s stdLogger) Criticalf(format string, v ...interface{}) {
	if !hasLevel(CRITICAL, s.disallow) {
		log.Printf("[CRITICAL] %s", fmt.Sprintf(format, v...))
	}

	os.Exit(1)
}
