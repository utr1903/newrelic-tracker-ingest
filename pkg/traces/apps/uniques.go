package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/utr1903/newrelic-tracker-ingest/internal/graphql"
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

type AppsUniques struct {
	Gqlc *graphql.GraphQlClient
}

func NewUniques() *AppsUniques {
	return &AppsUniques{
		Gqlc: graphql.NewGraphQlClient(queryTemplate),
	}
}

func (a *AppsUniques) Run() {
	accountId, err := strconv.ParseInt(os.Getenv("NEWRELIC_ACCOUNT_ID"), 10, 64)
	if err != nil {
		panic(err)
	}

	qv := &queryVariables{
		AccountId: accountId,
		NrqlQuery: "FROM Span SELECT uniques(entity.name) AS `apps` SINCE 1 week ago LIMIT MAX",
	}

	resBody, err := a.Gqlc.Execute(qv)
	if err != nil {
		panic(err)
	}

	res := &response{}
	err = json.Unmarshal(resBody, res)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Data.Actor.Nrql.Results[0])
}
