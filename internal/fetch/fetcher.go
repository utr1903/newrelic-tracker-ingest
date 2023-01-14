package fetch

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/utr1903/newrelic-tracker-ingest/internal/graphql"
	"github.com/utr1903/newrelic-tracker-ingest/internal/logging"
)

func FetchUniqueApps[T any](
	logger logging.ILogger,
	gqlc graphql.IGraphQlClient,
	qv any,
) (
	[]T,
	error,
) {
	res := &graphql.GraphQlResponse[T]{}
	err := gqlc.Execute(qv, res)
	if err != nil {
		return nil, err
	}

	if res.Errors != nil {
		logger.LogWithFields(logrus.DebugLevel, FETCHER_GRAPHQL_HAS_RETURNED_ERRORS,
			map[string]string{
				"tracker.package": "pkg.traces.apps",
				"tracker.file":    "uniques.go",
				"tracker.error":   fmt.Sprintf("%v", res.Errors),
			})
		return nil, errors.New(FETCHER_GRAPHQL_HAS_RETURNED_ERRORS)
	}

	return res.Data.Actor.Nrql.Results, nil
}