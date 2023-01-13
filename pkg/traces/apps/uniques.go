package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/utr1903/newrelic-tracker-ingest/internal/graphql"
	"github.com/utr1903/newrelic-tracker-ingest/internal/logging"
	"github.com/utr1903/newrelic-tracker-ingest/internal/metrics"
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
const trackedMetricName = "uniqueAppNames"

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

type UniquesApps struct {
	AccountId       int64
	Logger          *logging.Logger
	Gqlc            *graphql.GraphQlClient
	MetricForwarder *metrics.MetricForwarder
}

func NewUniqueApps(
	accountId int64,
) *UniquesApps {
	return &UniquesApps{
		AccountId: accountId,
		Logger: logging.NewLoggerWithForwarder(
			"DEBUG",
			os.Getenv("NEWRELIC_LICENSE_KEY"),
			"https://log-api.eu.newrelic.com/log/v1",
		),
		Gqlc: graphql.NewGraphQlClient(
			trackedMetricName,
			queryTemplate,
		),
		MetricForwarder: metrics.NewMetricForwarder(
			os.Getenv("NEWRELIC_LICENSE_KEY"),
			"https://metric-api.eu.newrelic.com/metric/v1",
		),
	}
}

func (a *UniquesApps) Run() error {

	// Fetch the unique application names per GraphQL
	appNames, err := a.fetchUniqueApps(a.AccountId)
	if err != nil {
		return nil
	}

	// Create & flush metrics
	a.flushMetrics(appNames)

	// Flush logs
	a.flushLogs()

	return nil
}

func (a *UniquesApps) fetchUniqueApps(
	accountId int64,
) (
	[]string,
	error,
) {
	qv := &queryVariables{
		AccountId: accountId,
		NrqlQuery: "FROM Span SELECT uniques(entity.name) AS `apps` SINCE 1 week ago LIMIT MAX",
	}

	resBody, err := a.Gqlc.Execute(qv)
	if err != nil {
		a.Logger.LogWithFields(logrus.ErrorLevel, APPS_UNIQUES_GRAPHQL_REQUEST_HAS_FAILED,
			map[string]string{
				"tracker.trackedMetricName": trackedMetricName,
				"tracker.accountId":         strconv.FormatInt(accountId, 10),
				"tracker.error":             err.Error(),
			})
		return nil, err
	}

	a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_PARSING_GRAPHQL_RESPONSE,
		map[string]string{
			"tracker.trackedMetricName": trackedMetricName,
			"tracker.accountId":         strconv.FormatInt(accountId, 10),
		})

	res := &response{}
	err = json.Unmarshal(resBody, res)
	if err != nil {
		a.Logger.LogWithFields(logrus.ErrorLevel, APPS_UNIQUES_GRAPHQL_RESPONSE_COULD_NOT_BE_PARSED,
			map[string]string{
				"tracker.trackedMetricName": trackedMetricName,
				"tracker.accountId":         strconv.FormatInt(accountId, 10),
				"tracker.error":             err.Error(),
			})
		return nil, err
	}

	return res.Data.Actor.Nrql.Results[0].Apps, nil
}

func (a *UniquesApps) flushMetrics(
	appNames []string,
) error {

	a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_FLUSHING_METRICS,
		map[string]string{
			"tracker.trackedMetricName": trackedMetricName,
			"tracker.accountId":         strconv.FormatInt(a.AccountId, 10),
		})

	// Add common block
	a.MetricForwarder.AddCommon(
		map[string]string{
			"tracker.trackedMetricName": trackedMetricName,
			"tracker.accountId":         strconv.FormatInt(a.AccountId, 10),
		},
	)

	// Add individual metrics
	for _, appName := range appNames {
		a.MetricForwarder.AddMetric(
			time.Now().UnixMicro(),
			"tracker.isActive",
			"gauge",
			1.0,
			map[string]string{
				"tracker.appName": appName,
			},
		)
	}

	err := a.MetricForwarder.Run()
	if err != nil {
		a.Logger.LogWithFields(logrus.ErrorLevel, APPS_UNIQUES_METRICS_COULD_NOT_BE_FORWARDED,
			map[string]string{
				"tracker.trackedMetricName": trackedMetricName,
				"tracker.accountId":         strconv.FormatInt(a.AccountId, 10),
				"tracker.error":             err.Error(),
			})
		return err
	}

	a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_METRICS_ARE_FORWARDED,
		map[string]string{
			"tracker.trackedMetricName": trackedMetricName,
			"tracker.accountId":         strconv.FormatInt(a.AccountId, 10),
		})

	return nil
}

func (a *UniquesApps) flushLogs() {
	err := a.Logger.Flush()
	if err != nil {
		fmt.Println(APPS_UNIQUES_LOGS_COULD_NOT_BE_FORWARDED, err.Error())
	}
}
