package graphql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const queryTemplate = `
{
  actor {
    nrql(
			accounts: {{ .AccountId }},
			query: "{{ .NrqlQuery }}"
		) {
      results
    }
  }
}
`

type queryVariablesMock struct {
	AccountId int64
	NrqlQuery string
}

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

type graphqlResponseMock struct {
	Result string
}

func Test_SubstituteTemplateQueryFails(t *testing.T) {
	logger := newLoggerMock()

	gqlc := NewGraphQlClient(
		logger,
		"newrelicGraphQlEndpoint",
		"test",
		queryTemplate,
	)
	_, err := gqlc.substituteTemplateQuery("wrong_template_vars")
	assert.NotNil(t, err)
}

func Test_SubstituteTemplateQuerySucceeds(t *testing.T) {
	accountId := int64(12345)
	nrqlQuery := "NRQL query"

	logger := newLoggerMock()

	gqlc := NewGraphQlClient(
		logger,
		"newrelicGraphQlEndpoint",
		"test",
		queryTemplate,
	)
	_, err := gqlc.substituteTemplateQuery(&queryVariablesMock{
		AccountId: accountId,
		NrqlQuery: nrqlQuery,
	})
	assert.Nil(t, err)
}

func Test_CreatePayloadSucceeds(t *testing.T) {
	logger := newLoggerMock()

	var query string = ""
	gqlc := NewGraphQlClient(
		logger,
		"newrelicGraphQlEndpoint",
		"test",
		queryTemplate,
	)
	buf, _ := gqlc.createPayload(&query)
	assert.NotNil(t, buf)
}

func Test_CreatingHttpRequestFails(t *testing.T) {
	accountId := int64(12345)
	nrqlQuery := "NRQL query"

	logger := newLoggerMock()

	res := &GraphQlResponse[string]{}
	gqlc := NewGraphQlClient(
		logger,
		"::",
		"test",
		queryTemplate,
	)
	err := gqlc.Execute(
		&queryVariablesMock{
			AccountId: accountId,
			NrqlQuery: nrqlQuery,
		},
		res)

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, GRAPHQL_CREATING_HTTP_REQUEST_HAS_FAILED)
}

func Test_PerformingHttpRequestFails(t *testing.T) {
	accountId := int64(12345)
	nrqlQuery := "NRQL query"

	logger := newLoggerMock()

	res := &GraphQlResponse[string]{}
	gqlc := NewGraphQlClient(
		logger,
		"",
		"test",
		queryTemplate,
	)
	err := gqlc.Execute(
		&queryVariablesMock{
			AccountId: accountId,
			NrqlQuery: nrqlQuery,
		},
		res)

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, GRAPHQL_PERFORMING_HTTP_REQUEST_HAS_FAILED)
}

func Test_ReadingHttpRequestFails(t *testing.T) {
	newrelicGraphQlServerMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var b bytes.Buffer
			w.WriteHeader(http.StatusBadRequest)
			w.Write(b.Bytes())
		}))
	defer newrelicGraphQlServerMock.Close()

	accountId := int64(12345)
	nrqlQuery := "NRQL query"

	logger := newLoggerMock()

	res := &GraphQlResponse[string]{}
	gqlc := NewGraphQlClient(
		logger,
		newrelicGraphQlServerMock.URL,
		"test",
		queryTemplate,
	)
	err := gqlc.Execute(
		&queryVariablesMock{
			AccountId: accountId,
			NrqlQuery: nrqlQuery,
		},
		res)

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, GRAPHQL_RESPONSE_HAS_RETURNED_NOT_OK_STATUS_CODE)
}

func Test_ParsingHttpRequestFails(t *testing.T) {
	newrelicGraphQlServerMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var b bytes.Buffer
			b.WriteString("error json")

			w.WriteHeader(http.StatusOK)
			w.Write(b.Bytes())
		}))
	defer newrelicGraphQlServerMock.Close()

	accountId := int64(12345)
	nrqlQuery := "NRQL query"

	logger := newLoggerMock()

	res := &GraphQlResponse[string]{}
	gqlc := NewGraphQlClient(
		logger,
		newrelicGraphQlServerMock.URL,
		"test",
		queryTemplate,
	)
	err := gqlc.Execute(
		&queryVariablesMock{
			AccountId: accountId,
			NrqlQuery: nrqlQuery,
		},
		res)

	assert.NotNil(t, err)
	assert.Contains(t, logger.msgs, GRAPHQL_PARSING_HTTP_RESPONSE_BODY_HAS_FAILED)
}

func Test_GraphQlRequestSucceeds(t *testing.T) {
	newrelicGraphQlServerMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var b bytes.Buffer

			res := &graphqlResponseMock{}
			json, _ := json.Marshal(res)
			b.Write(json)

			w.WriteHeader(http.StatusOK)
			w.Write(b.Bytes())
		}))
	defer newrelicGraphQlServerMock.Close()

	accountId := int64(12345)
	nrqlQuery := "NRQL query"

	logger := newLoggerMock()

	res := &GraphQlResponse[string]{}
	gqlc := NewGraphQlClient(
		logger,
		newrelicGraphQlServerMock.URL,
		"test",
		queryTemplate,
	)
	err := gqlc.Execute(
		&queryVariablesMock{
			AccountId: accountId,
			NrqlQuery: nrqlQuery,
		},
		res)

	assert.Nil(t, err)
}
