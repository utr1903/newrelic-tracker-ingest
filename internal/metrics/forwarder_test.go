package metrics

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type loggerMock struct {
	msgs []string
}

func newLoggerMock() *loggerMock {
	return &loggerMock{
		msgs: make([]string, 0),
	}
}
func (l *loggerMock) LogWithFields(
	lvl logrus.Level,
	msg string,
	attributes map[string]string,
) {
	l.msgs = append(l.msgs, msg)
}

func (l *loggerMock) Flush() error {
	return nil
}

func Test_NoMetricsToSend(t *testing.T) {
	logger := newLoggerMock()

	mf := NewMetricForwarder(
		logger,
		"licenseKey",
		"metricsEndpoint",
		map[string]string{},
	)

	err := mf.Run()

	assert.Nil(t, err)
	assert.Contains(t, logger.msgs, METRICS_THERE_ARE_NO_METRICS_TO_SEND)
}

func Test_CreatePayloadSucceeds(t *testing.T) {
	logger := newLoggerMock()

	mf := NewMetricForwarder(
		logger,
		"licenseKey",
		"metricsEndpoint",
		map[string]string{},
	)

	_, err := mf.createPayload()

	assert.Nil(t, err)
}

func Test_CreatingHttpRequestFails(t *testing.T) {
	logger := newLoggerMock()

	mf := NewMetricForwarder(
		logger,
		"licenseKey",
		"::",
		map[string]string{},
	)

	mf.AddMetric(
		time.Now().UnixMicro(),
		"test",
		"gauge",
		1.0,
		map[string]string{},
	)
	err := mf.Run()

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, METRICS_HTTP_REQUEST_COULD_NOT_BE_CREATED)
}

func Test_PerformingHttpRequestFails(t *testing.T) {
	logger := newLoggerMock()

	mf := NewMetricForwarder(
		logger,
		"licenseKey",
		"",
		map[string]string{},
	)

	mf.AddMetric(
		time.Now().UnixMicro(),
		"test",
		"gauge",
		1.0,
		map[string]string{},
	)
	err := mf.Run()

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, METRICS_HTTP_REQUEST_HAS_FAILED)
}

func Test_MetricApiReturnsNotOkStatus(t *testing.T) {
	newrelicMetricApiServerMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var b bytes.Buffer
			w.WriteHeader(http.StatusBadRequest)
			w.Write(b.Bytes())
		}))
	defer newrelicMetricApiServerMock.Close()

	logger := newLoggerMock()

	mf := NewMetricForwarder(
		logger,
		"licenseKey",
		newrelicMetricApiServerMock.URL,
		map[string]string{},
	)

	mf.AddMetric(
		time.Now().UnixMicro(),
		"test",
		"gauge",
		1.0,
		map[string]string{},
	)
	err := mf.Run()

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, METRICS_NEW_RELIC_RETURNED_NOT_OK_STATUS)
}

func Test_MetricApiRequestSucceeds(t *testing.T) {
	newrelicMetricApiServerMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var b bytes.Buffer
			w.WriteHeader(http.StatusAccepted)
			w.Write(b.Bytes())
		}))
	defer newrelicMetricApiServerMock.Close()

	logger := newLoggerMock()

	mf := NewMetricForwarder(
		logger,
		"licenseKey",
		newrelicMetricApiServerMock.URL,
		map[string]string{},
	)

	mf.AddMetric(
		time.Now().UnixMicro(),
		"test",
		"gauge",
		1.0,
		map[string]string{},
	)
	err := mf.Run()

	assert.Nil(t, err)
}
