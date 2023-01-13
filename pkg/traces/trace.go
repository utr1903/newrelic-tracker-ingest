package traces

import (
	"os"
	"strconv"

	"github.com/utr1903/newrelic-tracker-ingest/pkg/traces/apps"
)

func Run() {
	appsUniques := apps.NewUniques()
	accountId, _ := strconv.ParseInt(os.Getenv("NEWRELIC_ACCOUNT_ID"), 10, 64)
	appsUniques.Run(accountId)
}
