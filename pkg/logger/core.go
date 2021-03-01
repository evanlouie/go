// Package logger is a wrapper package around logrus which correctly outputs
// to stdout/stderr depending on the function. Logrus only outputs to stderr.
package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// lock is a global mutex lock to gain control of logrus.<SetLevel|SetOutput>
var lock = sync.Mutex{}

// SetLevelDebug sets the standard logger level to Debug
func SetLevelDebug() {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetLevel(logrus.DebugLevel)
}

// SetLevelInfo sets the standard logger level to Info
func SetLevelInfo() {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetLevel(logrus.InfoLevel)
}

// Trace logs a message at level Trace to stdout.
func Trace(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Trace(args...)
}

// Tracef logs a message at level Trace to stdout.
func Tracef(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Tracef(format, args...)
}

// Traceln logs a message at level Trace to stdout.
func Traceln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Traceln(args...)
}

// Debug logs a message at level Debug to stdout.
func Debug(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Debug(args...)
}

// Debugf logs a message at level Debug to stdout.
func Debugf(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Debugf(format, args...)
}

// Debugln logs a message at level Debug to stdout.
func Debugln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Debugln(args...)
}

// Info logs a message at level Info to stdout.
func Info(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Info(args...)
}

// Infof logs a message at level Info to stdout.
func Infof(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Infof(format, args...)
}

// Infoln logs a message at level Info to stdout.
func Infoln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Infoln(args...)
}

// Warn logs a message at level Warn to stdout.
func Warn(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Warn(args...)
}

// Warnf logs a message at level Warn to stdout.
func Warnf(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Warnf(format, args...)
}

// Warnln logs a message at level Warn to stdout.
func Warnln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stdout)
	logrus.Warnln(args...)
}

// Error logs a message at level Error to stderr.
func Error(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Error(args...)
}

// Errorf logs a message at level Error to stdout.
func Errorf(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Errorf(format, args...)
}

// Errorln logs a message at level Error to stdout.
func Errorln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Errorln(args...)
}

// Fatal logs a message at level Fatal to stderr then the process will exit with status set to 1.
func Fatal(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Fatal(args...)
}

// Fatalf logs a message at level Fatal to stdout.
func Fatalf(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Fatalf(format, args...)
}

// Fatalln logs a message at level Fatal to stdout.
func Fatalln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Fatalln(args...)
}

// Panic logs a message at level Panic to stderr; calls panic() after logging.
func Panic(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Panic(args...)
}

// Panicf logs a message at level Panic to stdout.
func Panicf(format string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Panicf(format, args...)
}

// Panicln logs a message at level Panic to stdout.
func Panicln(args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	logrus.SetOutput(os.Stderr)
	logrus.Panicln(args...)
}

// Echo is a general helper function to output sequential and indent based
// user feedback.
func Echo(level int, message interface{}) {
	decorator := "-"
	switch level {
	case 0:
		decorator = ">"
	case 1:
		decorator = "\u2192" // right arrow
	case 2:
		decorator = "+"
	}
	indent := strings.Repeat("\t", level)
	formattedMessage := fmt.Sprintf("%v%v %v\n", indent, decorator, message)
	if _, isError := message.(error); isError {
		Fatal(formattedMessage)
	}
	Info(formattedMessage)
}

func init() {
	// Setup logger defaults
	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = "02-01-2006 15:04:05"
	formatter.FullTimestamp = true
	logrus.SetFormatter(formatter)
	logrus.SetOutput(os.Stdout) // Set output to stdout; set to stderr by default
	logrus.SetLevel(logrus.InfoLevel)
}
