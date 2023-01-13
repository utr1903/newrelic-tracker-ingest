package metrics

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/utr1903/newrelic-tracker-ingest/internal/logging"
)

type commonBlock struct {
	Attributes map[string]string `json:"attributes"`
}

type metricBlock struct {
	Timestamp  int64             `json:"timestamp"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Value      float64           `json:"value"`
	Attributes map[string]string `json:"attributes"`
}

type metricObject struct {
	Common  *commonBlock  `json:"common"`
	Metrics []metricBlock `json:"metrics"`
}

type MetricForwarder struct {
	Logger           *logging.Logger
	MetricObjects    []metricObject
	client           *http.Client
	licenseKey       string
	metricsEndpoint  string
	commonAttributes map[string]string
}

func NewMetricForwarder(
	logger *logging.Logger,
	licenseKey string,
	metricsEndpoint string,
	commonAttributes map[string]string,
) *MetricForwarder {
	return &MetricForwarder{
		Logger: logger,
		MetricObjects: []metricObject{{
			Common: &commonBlock{
				Attributes: commonAttributes,
			},
			Metrics: []metricBlock{},
		}},
		client:           &http.Client{Timeout: time.Duration(30 * time.Second)},
		licenseKey:       licenseKey,
		metricsEndpoint:  metricsEndpoint,
		commonAttributes: commonAttributes,
	}
}

func (mf *MetricForwarder) AddMetric(
	metricTimestamp int64,
	metricName string,
	metricType string,
	metricValue float64,
	metricAttributes map[string]string,
) {
	mf.MetricObjects[0].Metrics = append(
		mf.MetricObjects[0].Metrics,
		metricBlock{
			Timestamp:  metricTimestamp,
			Name:       metricName,
			Type:       metricType,
			Value:      metricValue,
			Attributes: metricAttributes,
		},
	)
}

func (mf *MetricForwarder) Run() error {

	if len(mf.MetricObjects[0].Metrics) == 0 {
		mf.Logger.LogWithFields(logrus.DebugLevel, METRICS_THERE_ARE_NO_METRICS_TO_SEND,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
			})
		return nil
	}

	// Create zipped payload
	payloadZipped, err := mf.createPayload()
	if err != nil {
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest(
		http.MethodPost,
		mf.metricsEndpoint,
		payloadZipped,
	)
	if err != nil {
		mf.Logger.LogWithFields(logrus.ErrorLevel, METRICS_HTTP_REQUEST_COULD_NOT_BE_CREATED,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
				"tracker.error":   err.Error(),
			})
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Api-Key", mf.licenseKey)

	// Perform HTTP request
	res, err := mf.client.Do(req)
	if err != nil {
		mf.Logger.LogWithFields(logrus.ErrorLevel, METRICS_HTTP_REQUEST_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
				"tracker.error":   err.Error(),
			})
		return err
	}
	defer res.Body.Close()

	// Check if call was successful
	if res.StatusCode != http.StatusAccepted {
		mf.Logger.LogWithFields(logrus.ErrorLevel, METRICS_NEW_RELIC_RETURNED_NOT_OK_STATUS,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
				"tracker.error":   err.Error(),
			})
		return err
	}

	mf.Logger.LogWithFields(logrus.DebugLevel, METRICS_METRICS_ARE_FORWARDED,
		map[string]string{
			"tracker.package": "internal.metrics.metrics",
			"tracker.file":    "forwarder.go",
		})

	return nil
}

func (mf *MetricForwarder) createPayload() (
	*bytes.Buffer,
	error,
) {
	// Create payload
	mf.Logger.LogWithFields(logrus.DebugLevel, METRICS_CREATING_PAYLOAD,
		map[string]string{
			"tracker.package": "internal.metrics.metrics",
			"tracker.file":    "forwarder.go",
		})

	json, err := json.Marshal(mf.MetricObjects)
	if err != nil {
		mf.Logger.LogWithFields(logrus.ErrorLevel, METRICS_PAYLOAD_COULD_NOT_BE_CREATED,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
				"tracker.error":   err.Error(),
			})
		return nil, err
	}

	// Zip the payload
	var payloadZipped bytes.Buffer
	zw := gzip.NewWriter(&payloadZipped)
	defer zw.Close()

	if _, err = zw.Write(json); err != nil {
		mf.Logger.LogWithFields(logrus.ErrorLevel, METRICS_PAYLOAD_COULD_NOT_BE_ZIPPED,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
				"tracker.error":   err.Error(),
			})
		return nil, err
	}

	if err = zw.Close(); err != nil {
		mf.Logger.LogWithFields(logrus.ErrorLevel, METRICS_PAYLOAD_COULD_NOT_BE_ZIPPED,
			map[string]string{
				"tracker.package": "internal.metrics.metrics",
				"tracker.file":    "forwarder.go",
				"tracker.error":   err.Error(),
			})
		return nil, err
	}

	return &payloadZipped, nil
}
