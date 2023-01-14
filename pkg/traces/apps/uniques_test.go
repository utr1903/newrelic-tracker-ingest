package apps

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/utr1903/newrelic-tracker-ingest/internal/graphql"
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

var apps = []string{"app1", "app2"}

type graphqlClientMock struct {
	failRequest bool
}

func (c *graphqlClientMock) Execute(
	queryVariables any,
	result any,
) error {
	if c.failRequest {
		return errors.New("error")
	}

	res := result.(*graphql.GraphQlResponse[appNames])
	res.Data.Actor.Nrql.Results = []appNames{{
		Apps: apps,
	}}
	res.Errors = nil
	return nil
}

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

func Test_FetchingFails(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: true,
	}
	mf := &metricForwarderMock{
		returnError: true,
	}

	uas := &UniquesApps{
		AccountId:       int64(12345),
		Logger:          logger,
		Gqlc:            gqlc,
		MetricForwarder: mf,
	}

	err := uas.Run()

	assert.NotNil(t, err)
}

func Test_FetchingSucceeds(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: false,
	}
	mf := &metricForwarderMock{
		returnError: true,
	}

	uas := &UniquesApps{
		AccountId:       int64(12345),
		Logger:          logger,
		Gqlc:            gqlc,
		MetricForwarder: mf,
	}

	appNames, err := uas.fetchUniqueApps()

	assert.Nil(t, err)
	for i, appName := range appNames {
		assert.Equal(t, apps[i], appName)
	}
}

func Test_FlushingFails(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: false,
	}
	mf := &metricForwarderMock{
		returnError: true,
	}

	uas := &UniquesApps{
		AccountId:       int64(12345),
		Logger:          logger,
		Gqlc:            gqlc,
		MetricForwarder: mf,
	}

	err := uas.Run()

	assert.NotNil(t, err)
}

func Test_FlushingSucceeds(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: false,
	}
	mf := &metricForwarderMock{
		returnError: false,
	}

	uas := &UniquesApps{
		AccountId:       int64(12345),
		Logger:          logger,
		Gqlc:            gqlc,
		MetricForwarder: mf,
	}

	err := uas.Run()

	assert.Nil(t, err)
}
