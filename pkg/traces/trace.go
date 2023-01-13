package traces

import (
	"os"
	"strconv"
	"sync"

	"github.com/utr1903/newrelic-tracker-ingest/pkg/traces/apps"
	"github.com/utr1903/newrelic-tracker-ingest/pkg/traces/data"
)

func Run() {
	accountId, _ := strconv.ParseInt(os.Getenv("NEWRELIC_ACCOUNT_ID"), 10, 64)
	wg := new(sync.WaitGroup)

	// Unique applications
	wg.Add(1)
	go uniqueApps(wg, accountId)

	// Data ingests
	wg.Add(1)
	go dataIngests(wg, accountId)

	wg.Wait()
}

func uniqueApps(
	wg *sync.WaitGroup,
	accountId int64,
) {
	defer wg.Done()
	uniqueApps := apps.NewUniqueApps(accountId)
	uniqueApps.Run()
}

func dataIngests(
	wg *sync.WaitGroup,
	accountId int64,
) {
	defer wg.Done()
	dataIngests := data.NewDataIngests(accountId)
	dataIngests.Run()
}
