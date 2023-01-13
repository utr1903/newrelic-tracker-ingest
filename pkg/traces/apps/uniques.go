package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/utr1903/newrelic-tracker-ingest/internal/graphql"
	"github.com/utr1903/newrelic-tracker-ingest/internal/logging"
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
const trackedMetricName = "unique_app_names"

type queryVariables struct {
	AccountId int64
	NrqlQuery string
}

type result struct {
	Apps []string `json:"apps"`
}

type nrql struct {
	Results []result `json:"results"`
}

type actor struct {
	Nrql nrql `json:"nrql"`
}

type data struct {
	Actor actor `json:"actor"`
}

type response struct {
	Data data `json:"data"`
}

type AppsUniques struct {
	Logger *logging.Logger
	Gqlc   *graphql.GraphQlClient
}

func NewUniques() *AppsUniques {
	return &AppsUniques{
		Logger: logging.NewLoggerWithForwarder(
			"DEBUG",
			os.Getenv("NEWRELIC_LICENSE_KEY"),
			"https://log-api.eu.newrelic.com/log/v1",
		),
		Gqlc: graphql.NewGraphQlClient(trackedMetricName, queryTemplate),
	}
}

func (a *AppsUniques) Run(
	accountId int64,
) error {
	qv := &queryVariables{
		AccountId: accountId,
		NrqlQuery: "FROM Span SELECT uniques(entity.name) AS `apps` SINCE 1 week ago LIMIT MAX",
	}

	resBody, err := a.Gqlc.Execute(qv)
	if err != nil {
		a.Logger.LogWithFields(logrus.ErrorLevel, APPS_UNIQUES_GRAPHQL_REQUEST_HAS_FAILED,
			map[string]string{
				"trackedMetricName": trackedMetricName,
				"accountId":         strconv.FormatInt(accountId, 10),
				"error":             err.Error(),
			})
		return err
	}

	a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_PARSING_GRAPHQL_RESPONSE,
		map[string]string{
			"trackedMetricName": trackedMetricName,
			"accountId":         strconv.FormatInt(accountId, 10),
		})
	res := &response{}
	err = json.Unmarshal(resBody, res)
	if err != nil {
		a.Logger.LogWithFields(logrus.ErrorLevel, APPS_UNIQUES_GRAPHQL_RESPONSE_COULD_NOT_BE_PARSED,
			map[string]string{
				"trackedMetricName": trackedMetricName,
				"accountId":         strconv.FormatInt(accountId, 10),
				"error":             err.Error(),
			})
		return err
	}
	fmt.Println(res.Data.Actor.Nrql.Results[0])

	a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_METRICS_ARE_FORWARDED,
		map[string]string{
			"trackedMetricName": trackedMetricName,
			"accountId":         strconv.FormatInt(accountId, 10),
		})

	err = a.Logger.Flush()
	if err != nil {
		fmt.Println(APPS_UNIQUES_LOGS_COULD_NOT_BE_FORWARDED, err.Error())
		return err
	}

	return nil
}
