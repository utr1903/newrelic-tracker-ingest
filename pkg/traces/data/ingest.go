package data

import (
	"fmt"
	"os"
	"strconv"

	fetch "github.com/utr1903/newrelic-tracker-internal/fetch"
	flush "github.com/utr1903/newrelic-tracker-internal/flush"
	graphql "github.com/utr1903/newrelic-tracker-internal/graphql"
	logging "github.com/utr1903/newrelic-tracker-internal/logging"
	metrics "github.com/utr1903/newrelic-tracker-internal/metrics"
)

const (
	APPS_INGESTS_LOGS_COULD_NOT_BE_FORWARDED = "logs could not be forwarded"
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
	Logger          logging.ILogger
	Gqlc            graphql.IGraphQlClient
	MetricForwarder metrics.IMetricForwarder
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
		NrqlQuery: "FROM Span, ErrorTrace, SqlTrace SELECT bytecountestimate()/10e8 AS `ingest` WHERE instrumentation.provider != `pixie` FACET entity.name AS `app` SINCE 1 week ago LIMIT MAX",
	}

	apps, err := fetch.Fetch[appIngest](
		d.Logger,
		d.Gqlc,
		qv,
	)
	if err != nil {
		return nil, err
	}

	return apps, nil
}

func (d *DataIngest) flushMetrics(
	appIngests []appIngest,
) error {
	metrics := []flush.FlushMetric{}
	for _, appIngest := range appIngests {
		metrics = append(metrics, flush.FlushMetric{
			Name:  "tracker.dataIngest",
			Value: appIngest.Ingest,
			Attributes: map[string]string{
				"tracker.appName": appIngest.App,
			},
		})
	}
	err := flush.Flush(d.MetricForwarder, metrics)
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
