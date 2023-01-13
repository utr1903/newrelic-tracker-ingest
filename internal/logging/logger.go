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

type ILogger interface {
	LogWithFields(
		lvl logrus.Level,
		msg string,
		attributes map[string]string,
	)

	Flush() error
}

type Logger struct {
	log       *logrus.Logger
	forwarder *forwarder
}

func NewLoggerWithForwarder(
	logLevel string,
	licenseKey string,
	logsEndpoint string,
	commonAttributes map[string]string,
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

	f := newForwarder(
		logrus.AllLevels,
		licenseKey,
		logsEndpoint,
		commonAttributes,
	)

	l.AddHook(f)

	return &Logger{
		log:       l,
		forwarder: f,
	}
}

func (l *Logger) LogWithFields(
	lvl logrus.Level,
	msg string,
	attributes map[string]string,
) {

	fields := logrus.Fields{}

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

func (l *Logger) Flush() error {
	return l.forwarder.flush()
}
