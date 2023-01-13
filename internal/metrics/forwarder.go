package metrics

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http"
	"time"
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
	MetricObjects   []metricObject
	client          *http.Client
	licenseKey      string
	metricsEndpoint string
}

func NewMetricForwarder(
	licenseKey string,
	metricsEndpoint string,
) *MetricForwarder {
	return &MetricForwarder{
		MetricObjects: []metricObject{{
			Common:  &commonBlock{},
			Metrics: []metricBlock{},
		}},
		client:          &http.Client{Timeout: time.Duration(30 * time.Second)},
		licenseKey:      licenseKey,
		metricsEndpoint: metricsEndpoint,
	}
}

func (mf *MetricForwarder) AddCommon(
	attributes map[string]string,
) {
	mf.MetricObjects[0].Common = &commonBlock{
		Attributes: attributes,
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

	// Create zipped payload
	payloadZipped, err := mf.createPayload()
	if err != nil {
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, mf.metricsEndpoint, payloadZipped)
	if err != nil {
		return errors.New(METRICS_HTTP_REQUEST_COULD_NOT_BE_CREATED)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Api-Key", mf.licenseKey)

	// Perform HTTP request
	res, err := mf.client.Do(req)
	if err != nil {
		return errors.New(METRICS_HTTP_REQUEST_HAS_FAILED)
	}
	defer res.Body.Close()

	// Check if call was successful
	if res.StatusCode != http.StatusAccepted {
		return errors.New(METRICS_NEW_RELIC_RETURNED_NOT_OK_STATUS)
	}

	return nil
}

func (mf *MetricForwarder) createPayload() (
	*bytes.Buffer,
	error,
) {
	// Create payload
	json, err := json.Marshal(mf.MetricObjects)
	if err != nil {
		return nil, errors.New(METRICS_PAYLOAD_COULD_NOT_BE_CREATED)
	}

	// Zip the payload
	var payloadZipped bytes.Buffer
	zw := gzip.NewWriter(&payloadZipped)
	defer zw.Close()

	if _, err = zw.Write(json); err != nil {
		return nil, errors.New(METRICS_PAYLOAD_COULD_NOT_BE_ZIPPED)
	}

	if err = zw.Close(); err != nil {
		return nil, errors.New(METRICS_PAYLOAD_COULD_NOT_BE_ZIPPED)
	}

	return &payloadZipped, nil
}
