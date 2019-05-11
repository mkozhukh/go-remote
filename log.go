package remote

import "fmt"

var log Logger = defaultLogger{}

// Logger object, logrus interface is used by default
type Logger interface {
	Errorf(string, ...interface{})
	Debugf(string, ...interface{})
}

type defaultLogger struct{}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}

func (l defaultLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("DEBUG: "+format+"\n", args...)
}

// SetLogger allows to set default package logger
func SetLogger(logger Logger) {
	log = logger
}
