package graphql

// --- GraphQL for NRQL query --- //
type GraphQlResponse[T interface{}] struct {
	Data   data[T]     `json:"data"`
	Errors interface{} `json:"errors"`
}

type data[T interface{}] struct {
	Actor actor[T] `json:"actor"`
}

type actor[T interface{}] struct {
	Nrql nrql[T] `json:"nrql"`
}

type nrql[T interface{}] struct {
	Results []T `json:"results"`
}
