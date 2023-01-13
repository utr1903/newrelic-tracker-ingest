package graphql

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/utr1903/newrelic-tracker-ingest/internal/logging"
)

type graphQlRequestPayload struct {
	Query string `json:"query"`
}

type GraphQlClient struct {
	Logger            *logging.Logger
	HttpClient        *http.Client
	QueryTemplateName string
	QueryTemplate     string
}

func NewGraphQlClient(
	logger *logging.Logger,
	queryTemplateName string,
	queryTemplate string,
) *GraphQlClient {
	return &GraphQlClient{
		Logger:            logger,
		HttpClient:        &http.Client{Timeout: time.Duration(30 * time.Second)},
		QueryTemplateName: queryTemplateName,
		QueryTemplate:     queryTemplate,
	}
}

func (c *GraphQlClient) Execute(
	queryVariables any,
	result any,
) error {

	// Substitute variables within query
	query, err := c.substituteTemplateQuery(queryVariables)
	if err != nil {
		return err
	}

	// Create payload
	payload, err := c.createPayload(query)
	if err != nil {
		return err
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.eu.newrelic.com/graphql",
		payload,
	)
	if err != nil {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_CREATING_HTTP_REQUEST_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return err
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Api-Key", os.Getenv("NEWRELIC_API_KEY"))

	// Perform HTTP request
	res, err := c.HttpClient.Do(req)
	if err != nil {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_PERFORMING_HTTP_REQUEST_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return err
	}
	defer res.Body.Close()

	// Read HTTP response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_READING_HTTP_RESPONSE_BODY_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return err
	}

	// Check if call was successful
	if res.StatusCode != http.StatusOK {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_RESPONSE_HAS_RETURNED_NOT_OK_STATUS_CODE,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   GRAPHQL_RESPONSE_HAS_RETURNED_NOT_OK_STATUS_CODE,
			})
		return errors.New(GRAPHQL_RESPONSE_HAS_RETURNED_NOT_OK_STATUS_CODE)
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_PARSING_HTTP_RESPONSE_BODY_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return err
	}

	return nil
}

func (c *GraphQlClient) substituteTemplateQuery(
	queryVariables any,
) (
	*string,
	error,
) {
	// Parse query template
	c.Logger.LogWithFields(logrus.DebugLevel, GRAPHQL_SUBSTITUTING_TEMPLATE_VARIABLES,
		map[string]string{
			"tracker.package": "internal.graphql",
			"tracker.file":    "client.go",
		})
	t, err := template.New(c.QueryTemplateName).Parse(c.QueryTemplate)
	if err != nil {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_SUBSTITUTING_TEMPLATE_VARIABLES_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return nil, err
	}

	// Write substituted query template into buffer
	c.Logger.LogWithFields(logrus.DebugLevel, GRAPHQL_EXECUTING_REQUEST,
		map[string]string{
			"tracker.package": "internal.graphql",
			"tracker.file":    "client.go",
		})
	buf := new(bytes.Buffer)
	err = t.Execute(buf, queryVariables)
	if err != nil {
		c.Logger.LogWithFields(logrus.ErrorLevel, GRAPHQL_SUBSTITUTING_TEMPLATE_VARIABLES_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return nil, err
	}

	// Return substituted query as string
	str := buf.String()
	return &str, nil
}

func (c *GraphQlClient) createPayload(
	query *string,
) (
	*bytes.Buffer,
	error,
) {

	// Create JSON data
	payload, err := json.Marshal(&graphQlRequestPayload{
		Query: *query,
	})
	if err != nil {
		c.Logger.LogWithFields(logrus.DebugLevel, GRAPHQL_CREATING_PAYLOAD_HAS_FAILED,
			map[string]string{
				"tracker.package": "internal.graphql",
				"tracker.file":    "client.go",
				"tracker.error":   err.Error(),
			})
		return nil, err
	}
	return bytes.NewBuffer(payload), nil
}
