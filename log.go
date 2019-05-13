package remote

import "fmt"

var log Logger = defaultLogger{}

// Logger object, logrus interface is used by default
type Logger interface {
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
	Debugf(string, ...interface{})
}

type defaultLogger struct{}

func (l defaultLogger) Fatalf(format string, args ...interface{}) {
	txt := fmt.Sprintf(format, args...)
	fmt.Print("ERROR: "+txt+"\n")
	panic(txt)
}

func (l defaultLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}

// SetLogger allows to set package logger
func SetLogger(logger Logger) {
	log = logger
}
