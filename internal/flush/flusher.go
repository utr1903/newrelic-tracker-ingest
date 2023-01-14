package flush

import (
	"time"

	"github.com/utr1903/newrelic-tracker-ingest/internal/metrics"
)

const (
	FLUSHER_FLUSHING_METRICS = "flushing metrics"
)

type FlushMetric struct {
	Name       string
	Value      float64
	Attributes map[string]string
}

func Flush(
	mf metrics.IMetricForwarder,
	metrics []FlushMetric,
) error {

	// Add individual metrics
	for _, metric := range metrics {
		mf.AddMetric(
			time.Now().UnixMicro(),
			metric.Name,
			"gauge",
			metric.Value,
			metric.Attributes,
		)
	}

	err := mf.Run()
	if err != nil {
		return err
	}

	return nil
}
