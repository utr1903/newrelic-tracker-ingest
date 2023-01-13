package traces

import (
	"os"
	"strconv"

	"github.com/utr1903/newrelic-tracker-ingest/pkg/traces/apps"
	"github.com/utr1903/newrelic-tracker-ingest/pkg/traces/data"
)

func Run() {
	accountId, _ := strconv.ParseInt(os.Getenv("NEWRELIC_ACCOUNT_ID"), 10, 64)

	// // Unique applications
	uniqueApps := apps.NewUniqueApps(accountId)
	uniqueApps.Run()

	// Data ingests
	dataIngests := data.NewDataIngests(accountId)
	dataIngests.Run()
}
