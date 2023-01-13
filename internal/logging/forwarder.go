package logging

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type commonBlock struct {
	Attributes map[string]string `json:"attributes"`
}

type logBlock struct {
	Timestamp  int64             `json:"timestamp"`
	Message    string            `json:"message"`
	Attributes map[string]string `json:"attributes"`
}

type logObject struct {
	Common *commonBlock `json:"common"`
	Logs   []logBlock   `json:"logs"`
}

type forwarder struct {
	levels []logrus.Level
	logs   []logrus.Entry

	client           *http.Client
	licenseKey       string
	logsEndpoint     string
	commonAttributes map[string]string
}

func newForwarder(
	levels []logrus.Level,
	licenseKey string,
	logsEndpoint string,
	commonAttributes map[string]string,
) *forwarder {
	return &forwarder{
		levels:           levels,
		logs:             make([]logrus.Entry, 0),
		client:           &http.Client{Timeout: time.Duration(30 * time.Second)},
		licenseKey:       licenseKey,
		logsEndpoint:     logsEndpoint,
		commonAttributes: setCommonAttributes(commonAttributes),
	}
}

func setCommonAttributes(
	commonAttrs map[string]string,
) map[string]string {

	// Copy the given attributes
	attrs := make(map[string]string)
	for k, v := range commonAttrs {
		attrs[k] = v
	}

	// Add the fixed attributes afterwards to avoid them
	// being overridden by the given attributes

	// Instrumentation provider
	attrs["instrumentation.provider"] = "newrelic-tracker-ingest"

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

func (f *forwarder) Levels() []logrus.Level {
	return f.levels
}

func (f *forwarder) Fire(e *logrus.Entry) error {
	copy := *e
	f.logs = append(f.logs, copy)
	return nil
}

func (f *forwarder) flush() error {
	// Return if there are no logs
	if len(f.logs) == 0 {
		return nil
	}

	// Create New Relic logs
	nrLogs := f.createNewRelicLogs()

	// Flush data to New Relic
	return f.sendToNewRelic(nrLogs)
}

func (f *forwarder) createNewRelicLogs() []logObject {
	lo := &logObject{
		Common: &commonBlock{
			Attributes: make(map[string]string),
		},
		Logs: make([]logBlock, 0, len(f.logs)),
	}

	// Create common block
	for key, val := range f.commonAttributes {
		lo.Common.Attributes[key] = val
	}

	// Create logs block
	for _, log := range f.logs {
		logBlock := logBlock{
			Timestamp:  log.Time.UnixMicro(),
			Message:    log.Message,
			Attributes: make(map[string]string),
		}

		for key, val := range log.Data {
			logBlock.Attributes[key] = fmt.Sprintf("%v", val)
		}
		lo.Logs = append(lo.Logs, logBlock)
	}

	return []logObject{*lo}
}

func (f *forwarder) sendToNewRelic(
	nrLogs []logObject,
) error {

	// Create zipped payload
	payloadZipped, err := f.createPayload(nrLogs)
	if err != nil {
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, f.logsEndpoint, payloadZipped)
	if err != nil {
		return errors.New(LOGS_HTTP_REQUEST_COULD_NOT_BE_CREATED)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Api-Key", f.licenseKey)

	// Perform HTTP request
	res, err := f.client.Do(req)
	if err != nil {
		return errors.New(LOGS_HTTP_REQUEST_HAS_FAILED)
	}
	defer res.Body.Close()

	// Check if call was successful
	if res.StatusCode != http.StatusAccepted {
		return errors.New(LOGS_NEW_RELIC_RETURNED_NOT_OK_STATUS)
	}

	return nil
}

func (f *forwarder) createPayload(
	nrLogs []logObject,
) (
	*bytes.Buffer,
	error,
) {
	// Create payload
	json, err := json.Marshal(nrLogs)
	if err != nil {
		return nil, errors.New(LOGS_PAYLOAD_COULD_NOT_BE_CREATED)
	}

	// Zip the payload

	var payloadZipped bytes.Buffer
	zw := gzip.NewWriter(&payloadZipped)
	defer zw.Close()

	if _, err = zw.Write(json); err != nil {
		return nil, errors.New(LOGS_PAYLOAD_COULD_NOT_BE_ZIPPED)
	}

	if err = zw.Close(); err != nil {
		return nil, errors.New(LOGS_PAYLOAD_COULD_NOT_BE_ZIPPED)
	}

	return &payloadZipped, nil
}
