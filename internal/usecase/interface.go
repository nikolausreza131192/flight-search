package usecase

import (
	"context"

	"bookcabin/internal/entity"
)

// SearchItf is the inbound port exposed to the delivery layer.
type SearchItf interface {
	Search(ctx context.Context, req SearchRequest) (SearchResponse, error)
}

// SearchResult is the merged output of a fan-out search across providers. It is
// part of the SearchRepository port contract, so it lives beside the port.
type SearchResult struct {
	Flights   []entity.Flight
	Queried   int
	Succeeded int
	Failed    int
	Errors    map[string]string
}

// SearchRepository is the outbound port implemented by the repository layer.
// It fans a criteria out to every provider and merges the results.
type SearchRepository interface {
	Search(ctx context.Context, criteria entity.SearchCriteria) SearchResult
}

// ResultCache is the outbound port used to memoize search responses.
type ResultCache interface {
	Get(key string) (SearchResponse, bool)
	Set(key string, value SearchResponse)
}
