package traces

import "github.com/utr1903/newrelic-tracker-ingest/pkg/traces/apps"

func Run() {
	appsUniques := apps.NewUniques()
	appsUniques.Run()
}
