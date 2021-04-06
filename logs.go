package go_remote

import (
	"fmt"
	stdlog "log"
)

var log Logger = defaultLogger{}

// SetLogger allows to set default package logger
func SetLogger(logger Logger) {
	log = logger
}

// Logger interface, used for error and debug logging
type Logger interface {
	Errorf(string, ...interface{})
	Debugf(string, ...interface{})
}

type defaultLogger struct{}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	stdlog.Printf("ERROR: "+format+"\n", args...)
}

func (l defaultLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("DEBUG: "+format+"\n", args...)
}
