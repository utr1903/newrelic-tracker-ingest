package apps

import (
	"errors"
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

const trackedAttributeType = "uniqueAppNames"

type queryVariables struct {
	AccountId int64
	NrqlQuery string
}

type appNames struct {
	Apps []string `json:"apps"`
}

type UniquesApps struct {
	AccountId       int64
	Logger          logging.ILogger
	Gqlc            graphql.IGraphQlClient
	MetricForwarder *metrics.MetricForwarder
}

func NewUniqueApps(
	accountId int64,
) *UniquesApps {
	logger := logging.NewLoggerWithForwarder(
		"DEBUG",
		os.Getenv("NEWRELIC_LICENSE_KEY"),
		"https://log-api.eu.newrelic.com/log/v1",
		setCommonAttributes(accountId),
	)
	gqlc := graphql.NewGraphQlClient(
		logger,
		"https://api.eu.newrelic.com/graphql",
		trackedAttributeType,
		queryTemplate,
	)
	mf := metrics.NewMetricForwarder(
		logger,
		os.Getenv("NEWRELIC_LICENSE_KEY"),
		"https://metric-api.eu.newrelic.com/metric/v1",
		setCommonAttributes(accountId),
	)
	return &UniquesApps{
		AccountId:       accountId,
		Logger:          logger,
		Gqlc:            gqlc,
		MetricForwarder: mf,
	}
}

func setCommonAttributes(
	accountId int64,
) map[string]string {
	return map[string]string{
		"tracker.attributeType": trackedAttributeType,
		"tracker.accountId":     strconv.FormatInt(accountId, 10),
	}
}

func (a *UniquesApps) Run() error {

	// Fetch the unique application names per GraphQL
	appNames, err := a.fetchUniqueApps()
	if err != nil {
		return nil
	}

	// Create & flush metrics
	err = a.flushMetrics(appNames)
	if err != nil {
		return nil
	}

	// Flush logs
	a.flushLogs()

	return nil
}

func (a *UniquesApps) fetchUniqueApps() (
	[]string,
	error,
) {
	qv := &queryVariables{
		AccountId: a.AccountId,
		NrqlQuery: "FROM Span SELECT uniques(entity.name) AS `apps` SINCE 1 week ago LIMIT MAX",
	}

	res := &graphql.GraphQlResponse[appNames]{}
	err := a.Gqlc.Execute(qv, res)
	if err != nil {
		return nil, err
	}

	if res.Errors != nil {
		a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_GRAPHQL_HAS_RETURNED_ERRORS,
			map[string]string{
				"tracker.package": "pkg.traces.apps",
				"tracker.file":    "uniques.go",
				"tracker.error":   fmt.Sprintf("%v", res.Errors),
			})
		return nil, errors.New(APPS_UNIQUES_GRAPHQL_HAS_RETURNED_ERRORS)
	}

	return res.Data.Actor.Nrql.Results[0].Apps, nil
}

func (a *UniquesApps) flushMetrics(
	appNames []string,
) error {

	a.Logger.LogWithFields(logrus.DebugLevel, APPS_UNIQUES_FLUSHING_METRICS,
		map[string]string{
			"tracker.package": "pkg.traces.apps",
			"tracker.file":    "uniques.go",
		})

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
		return err
	}

	return nil
}

func (a *UniquesApps) flushLogs() {
	err := a.Logger.Flush()
	if err != nil {
		fmt.Println(APPS_UNIQUES_LOGS_COULD_NOT_BE_FORWARDED, err.Error())
	}
}
