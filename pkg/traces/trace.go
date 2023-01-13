package traces

import (
	"os"
	"strconv"

	"github.com/utr1903/newrelic-tracker-ingest/pkg/traces/apps"
)

func Run() {
	accountId, _ := strconv.ParseInt(os.Getenv("NEWRELIC_ACCOUNT_ID"), 10, 64)
	appsUniques := apps.NewUniqueApps(accountId)
	appsUniques.Run()
}
