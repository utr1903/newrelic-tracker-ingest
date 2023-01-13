package data

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

const trackedAttributeType = "dataIngest"

type queryVariables struct {
	AccountId int64
	NrqlQuery string
}

type appIngest struct {
	App    string  `json:"app"`
	Ingest float64 `json:"ingest"`
}

type DataIngest struct {
	AccountId       int64
	Logger          *logging.Logger
	Gqlc            *graphql.GraphQlClient
	MetricForwarder *metrics.MetricForwarder
}

func NewDataIngests(
	accountId int64,
) *DataIngest {
	logger := logging.NewLoggerWithForwarder(
		"DEBUG",
		os.Getenv("NEWRELIC_LICENSE_KEY"),
		"https://log-api.eu.newrelic.com/log/v1",
		setCommonAttributes(accountId),
	)
	gqlc := graphql.NewGraphQlClient(
		logger,
		trackedAttributeType,
		queryTemplate,
	)
	mf := metrics.NewMetricForwarder(
		logger,
		os.Getenv("NEWRELIC_LICENSE_KEY"),
		"https://metric-api.eu.newrelic.com/metric/v1",
		setCommonAttributes(accountId),
	)
	return &DataIngest{
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

func (d *DataIngest) Run() error {

	// Fetch the unique application names per GraphQL
	appIngests, err := d.fetchDataIngets()
	if err != nil {
		return nil
	}

	// Create & flush metrics
	err = d.flushMetrics(appIngests)
	if err != nil {
		return nil
	}

	// Flush logs
	d.flushLogs()

	return nil
}

func (d *DataIngest) fetchDataIngets() (
	[]appIngest,
	error,
) {
	qv := &queryVariables{
		AccountId: d.AccountId,
		NrqlQuery: "FROM Span, ErrorTrace, SqlTrace SELECT bytecountestimate()/10e8 AS `ingest` WHERE instrumentation.provider != 'pixie' FACET entity.name AS `app` SINCE 1 week ago LIMIT MAX",
	}

	res := &graphql.GraphQlResponse[appIngest]{}
	err := d.Gqlc.Execute(qv, res)
	if err != nil {
		return nil, err
	}

	if res.Errors != nil {
		d.Logger.LogWithFields(logrus.DebugLevel, APPS_INGESTS_GRAPHQL_HAS_RETURNED_ERRORS,
			map[string]string{
				"tracker.package": "pkg.traces.data",
				"tracker.file":    "ingest.go",
				"tracker.error":   fmt.Sprintf("%v", res.Errors),
			})
		return nil, errors.New(APPS_INGESTS_GRAPHQL_HAS_RETURNED_ERRORS)
	}

	return res.Data.Actor.Nrql.Results, nil
}

func (d *DataIngest) flushMetrics(
	appIngests []appIngest,
) error {

	d.Logger.LogWithFields(logrus.DebugLevel, APPS_INGESTS_FLUSHING_METRICS,
		map[string]string{
			"tracker.package": "pkg.traces.data",
			"tracker.file":    "ingest.go",
		})

	// Add individual metrics
	for _, appIngest := range appIngests {
		d.MetricForwarder.AddMetric(
			time.Now().UnixMicro(),
			"tracker.dataIngest",
			"gauge",
			appIngest.Ingest,
			map[string]string{
				"tracker.appName": appIngest.App,
			},
		)
	}

	err := d.MetricForwarder.Run()
	if err != nil {
		return err
	}

	return nil
}

func (d *DataIngest) flushLogs() {
	err := d.Logger.Flush()
	if err != nil {
		fmt.Println(APPS_INGESTS_LOGS_COULD_NOT_BE_FORWARDED, err.Error())
	}
}
