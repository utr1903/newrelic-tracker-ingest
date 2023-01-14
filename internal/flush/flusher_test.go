package flush

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type metricForwarderMock struct {
	returnError bool
}

func (mf *metricForwarderMock) AddMetric(
	metricTimestamp int64,
	metricName string,
	metricType string,
	metricValue float64,
	metricAttributes map[string]string,
) {

}

func (mf *metricForwarderMock) Run() error {

	if mf.returnError {
		return errors.New("error")
	}
	return nil
}

func Test_MetricForwarderReturnsError(t *testing.T) {
	mf := &metricForwarderMock{
		returnError: true,
	}

	err := Flush(mf, []FlushMetric{})
	assert.NotNil(t, err)
}

func Test_MetricForwarderSucceeds(t *testing.T) {
	mf := &metricForwarderMock{
		returnError: false,
	}

	err := Flush(mf, []FlushMetric{})
	assert.Nil(t, err)
}
