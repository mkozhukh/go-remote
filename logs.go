package go_remote

import (
	"os"

	deflog "github.com/charmbracelet/log"
)

// Logger interface, used for error and debug logging
type Logger interface {
	Info(interface{}, ...interface{})
	Warn(interface{}, ...interface{})
	Fatal(interface{}, ...interface{})
	Error(interface{}, ...interface{})
	Debug(interface{}, ...interface{})
}

var log Logger

func init() {
	t := deflog.New(os.Stdout)
	t.SetLevel(deflog.DebugLevel)
	log = t
}

// SetLogger allows to set default package logger
func SetLogger(l Logger) {
	log = l
}
