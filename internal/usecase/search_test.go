package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bookcabin/config"
	"bookcabin/internal/entity"
	"bookcabin/internal/repository"
	"bookcabin/internal/usecase"
	"bookcabin/internal/util/cache"
)

func newTestUseCase(t *testing.T) *usecase.SearchUseCase {
	t.Helper()
	cfg, err := config.Load("../../template/config.json")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.CacheTTL = time.Minute
	cfg.SearchTimeout = 2 * time.Second
	cfg.MaxRetries = 2
	cfg.RetryBaseDelay = time.Millisecond
	cfg.SimulateLatency = false

	providers := repository.NewProviders(cfg)
	repo := repository.NewSearchRepo(providers, cfg)
	return usecase.New(cache.New[usecase.SearchResponse](cfg.CacheTTL), repo, usecase.Options{CacheTTL: cfg.CacheTTL})
}

func oneWay(origin, destination, date string) usecase.SearchRequest {
	return usecase.SearchRequest{Origin: origin, Destination: destination, DepartureDate: date, Passengers: 1, CabinClass: "economy"}
}

func TestSearchOneWayAggregatesAllProviders(t *testing.T) {
	svc := newTestUseCase(t)

	resp, err := svc.Search(context.Background(), oneWay("CGK", "DPS", "2025-12-15"))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Metadata.TotalResults != 13 {
		t.Fatalf("total results = %d, want 13", resp.Metadata.TotalResults)
	}
	if resp.Metadata.ProvidersQueried != 4 || resp.Metadata.ProvidersSucceeded != 4 || resp.Metadata.ProvidersFailed != 0 {
		t.Fatalf("unexpected provider metadata: %+v", resp.Metadata)
	}
	if resp.SearchCriteria.Origin != "CGK" || resp.SearchCriteria.DepartureDate != "2025-12-15" {
		t.Fatalf("unexpected search criteria: %+v", resp.SearchCriteria)
	}
	// Default ordering is best value, descending.
	for i := 1; i < len(resp.Flights); i++ {
		if resp.Flights[i].BestValueScore > resp.Flights[i-1].BestValueScore {
			t.Fatalf("results not sorted by best value at index %d", i)
		}
	}
}

func TestSearchIsCached(t *testing.T) {
	svc := newTestUseCase(t)
	req := oneWay("CGK", "DPS", "2025-12-15")

	if _, err := svc.Search(context.Background(), req); err != nil {
		t.Fatalf("first search: %v", err)
	}
	resp, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if !resp.Metadata.CacheHit {
		t.Fatal("expected second identical search to be served from cache")
	}
}

func TestSearchCacheHitReportsOwnDuration(t *testing.T) {
	cfg, err := config.Load("../../template/config.json")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.CacheTTL = time.Minute
	cfg.SearchTimeout = 5 * time.Second
	cfg.MaxRetries = 0
	cfg.SimulateLatency = true // slow the first search so the two durations differ

	repo := repository.NewSearchRepo(repository.NewProviders(cfg), cfg)
	svc := usecase.New(cache.New[usecase.SearchResponse](cfg.CacheTTL), repo, usecase.Options{CacheTTL: cfg.CacheTTL})

	req := oneWay("CGK", "DPS", "2025-12-15")
	first, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	second, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if !second.Metadata.CacheHit {
		t.Fatal("expected second search to be served from cache")
	}
	if second.Metadata.SearchTimeMs >= first.Metadata.SearchTimeMs {
		t.Fatalf("cache hit reported %dms, want less than the original %dms",
			second.Metadata.SearchTimeMs, first.Metadata.SearchTimeMs)
	}
}

func TestSearchFiltersDirectFlights(t *testing.T) {
	svc := newTestUseCase(t)
	req := oneWay("CGK", "DPS", "2025-12-15")
	zero := 0
	req.Filters = &usecase.Filters{MaxStops: &zero}

	resp, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Metadata.TotalResults != 9 {
		t.Fatalf("direct flight count = %d, want 9", resp.Metadata.TotalResults)
	}
	for _, f := range resp.Flights {
		if f.Stops != 0 {
			t.Fatalf("filter leaked a non-direct flight: %s (%d stops)", f.FlightNumber, f.Stops)
		}
	}
}

func TestSearchScalesPriceByPassengers(t *testing.T) {
	svc := newTestUseCase(t)
	req := oneWay("CGK", "DPS", "2025-12-15")
	req.Passengers = 2

	resp, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, f := range resp.Flights {
		if f.FlightNumber == "QZ532" && f.Price.Amount != 1190000 { // 595000 * 2
			t.Fatalf("QZ532 price = %d, want 1190000", f.Price.Amount)
		}
	}
}

func TestSearchRoundTrip(t *testing.T) {
	svc := newTestUseCase(t)
	ret := "2025-12-20"
	req := oneWay("CGK", "DPS", "2025-12-15")
	req.ReturnDate = &ret

	resp, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(resp.Itineraries) == 0 {
		t.Fatal("expected round trip itineraries")
	}
	first := resp.Itineraries[0]
	if first.DepartureJourney.Departure.Airport != "CGK" || first.ReturnJourney.Departure.Airport != "DPS" {
		t.Fatalf("unexpected itinerary endpoints: %+v", first)
	}
	if first.TotalPrice.Amount != first.DepartureJourney.Price.Amount+first.ReturnJourney.Price.Amount {
		t.Fatalf("itinerary price not summed: %+v", first.TotalPrice)
	}
}

func TestSearchMultiCity(t *testing.T) {
	svc := newTestUseCase(t)
	req := usecase.SearchRequest{
		Passengers: 1,
		CabinClass: "economy",
		MultiCity: []usecase.MultiCityLeg{
			{Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15"},
			{Origin: "DPS", Destination: "CGK", DepartureDate: "2025-12-20"},
		},
	}

	resp, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(resp.Legs) != 2 {
		t.Fatalf("legs = %d, want 2", len(resp.Legs))
	}
	for _, leg := range resp.Legs {
		if len(leg.Flights) == 0 {
			t.Fatalf("leg %s->%s returned no flights", leg.SearchCriteria.Origin, leg.SearchCriteria.Destination)
		}
	}
}

func TestSearchValidation(t *testing.T) {
	svc := newTestUseCase(t)
	_, err := svc.Search(context.Background(), usecase.SearchRequest{Origin: "CGK", DepartureDate: "2025-12-15"})
	if !errors.Is(err, entity.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
