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
)

const UNIQUE_APPLICATION_NAMES = "unique_app_names"

type graphQlRequestPayload struct {
	Query string `json:"query"`
}

type GraphQlClient struct {
	HttpClient    *http.Client
	QueryTemplate string
}

func NewGraphQlClient(
	queryTemplate string,
) *GraphQlClient {
	return &GraphQlClient{
		HttpClient:    &http.Client{Timeout: time.Duration(30 * time.Second)},
		QueryTemplate: queryTemplate,
	}
}

func (c *GraphQlClient) Execute(
	queryVariables any,
) (
	[]byte,
	error,
) {

	// Substitute variables within query
	query, err := c.substituteTemplateQuery(queryVariables)
	if err != nil {
		return nil, err
	}

	payload, err := c.createPayload(query)
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequest(http.MethodPost, "https://api.eu.newrelic.com/graphql", payload)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Api-Key", os.Getenv("NEWRELIC_API_KEY"))

	// Perform HTTP request
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read HTTP response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Check if call was successful
	if res.StatusCode != http.StatusOK {
		return body, errors.New("graphql request returned an error")
	}

	return body, nil
}

func (c *GraphQlClient) substituteTemplateQuery(
	queryVariables any,
) (
	*string,
	error,
) {
	// Parse query template
	t, err := template.New(UNIQUE_APPLICATION_NAMES).Parse(c.QueryTemplate)
	if err != nil {
		return nil, err
	}

	// Write substituted query template into buffer
	buf := new(bytes.Buffer)
	err = t.Execute(buf, queryVariables)
	if err != nil {
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
		return nil, err
	}
	return bytes.NewBuffer(payload), nil
}
