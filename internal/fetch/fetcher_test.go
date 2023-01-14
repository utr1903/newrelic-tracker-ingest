package fetch

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

type graphqlClientMock struct {
	failRequest bool
	returnError bool
}

func (c *graphqlClientMock) Execute(
	queryVariables any,
	result any,
) error {
	if c.failRequest {
		return errors.New("error")
	}
	if c.returnError {
		res := result.(*graphql.GraphQlResponse[string])
		res.Errors = []string{"err"}
		return nil
	}
	res := result.(*graphql.GraphQlResponse[string])
	res.Data.Actor.Nrql.Results = []string{"data"}
	res.Errors = nil
	return nil
}

func Test_GraphQlRequestFails(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: true,
	}

	res, err := Fetch[string](logger, gqlc, "qv")
	assert.Nil(t, res)
	assert.NotNil(t, err)
}

func Test_GraphQlRequestReturnsError(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: false,
		returnError: true,
	}

	res, err := Fetch[string](logger, gqlc, "qv")
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, FETCHER_GRAPHQL_HAS_RETURNED_ERRORS)
}

func Test_GraphQlRequestSucceeds(t *testing.T) {
	logger := newLoggerMock()
	gqlc := &graphqlClientMock{
		failRequest: false,
		returnError: false,
	}

	res, err := Fetch[string](logger, gqlc, "qv")
	assert.NotNil(t, res)
	assert.Nil(t, err)
}
