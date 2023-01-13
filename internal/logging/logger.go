package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

const (
	LOGS_PAYLOAD_COULD_NOT_BE_CREATED      = "payload could not be created"
	LOGS_PAYLOAD_COULD_NOT_BE_ZIPPED       = "payload could not be zipped"
	LOGS_HTTP_REQUEST_COULD_NOT_BE_CREATED = "http request could not be created"
	LOGS_HTTP_REQUEST_HAS_FAILED           = "http request has failed"
	LOGS_NEW_RELIC_RETURNED_NOT_OK_STATUS  = "http request has returned not OK status"
)

type Logger struct {
	log       *logrus.Logger
	forwarder *forwarder
}

func NewLogger(
	logLevel string,
) *Logger {
	l := logrus.New()
	l.Out = os.Stdout
	l.Formatter = &logrus.JSONFormatter{}

	switch logLevel {
	case "DEBUG":
		l.Level = logrus.DebugLevel
	default:
		l.Level = logrus.ErrorLevel
	}

	return &Logger{
		log:       l,
		forwarder: nil,
	}
}

func NewLoggerWithForwarder(
	logLevel string,
	licenseKey string,
	logsEndpoint string,
) *Logger {
	l := logrus.New()
	l.Out = os.Stdout
	l.Formatter = &logrus.JSONFormatter{}

	switch logLevel {
	case "DEBUG":
		l.Level = logrus.DebugLevel
	default:
		l.Level = logrus.ErrorLevel
	}

	f := newForwarder(logrus.AllLevels, licenseKey, logsEndpoint)
	l.AddHook(f)

	return &Logger{
		log:       l,
		forwarder: f,
	}
}

func (l *Logger) Log(
	lvl logrus.Level,
	msg string,
) {

	fields := logrus.Fields{}

	// Put common attributes
	for key, val := range getCommonAttributes() {
		fields[key] = val
	}

	switch lvl {
	case logrus.ErrorLevel:
		l.log.WithFields(fields).Error(msg)
	default:
		l.log.WithFields(fields).Debug(msg)
	}
}

func (l *Logger) LogWithFields(
	lvl logrus.Level,
	msg string,
	attributes map[string]string,
) {

	fields := logrus.Fields{}

	// Put common attributes
	for key, val := range getCommonAttributes() {
		fields[key] = val
	}

	// Put specific attributes
	for key, val := range attributes {
		fields[key] = val
	}

	switch lvl {
	case logrus.ErrorLevel:
		l.log.WithFields(fields).Error(msg)
	default:
		l.log.WithFields(fields).Debug(msg)
	}
}

func getCommonAttributes() map[string]string {
	attrs := map[string]string{
		"instrumentation.provider": "newrelic-tracker-ingest",
	}
	// Node name
	if val := os.Getenv("NODE_NAME"); val != "" {
		attrs["nodeName"] = val
	}

	// Namespace name
	if val := os.Getenv("NAMESPACE_NAME"); val != "" {
		attrs["namespaceName"] = val
	}

	// Pod name
	if val := os.Getenv("POD_NAME"); val != "" {
		attrs["podName"] = val
	}
	return attrs
}

func (l *Logger) Flush() error {
	return l.forwarder.flush()
}
