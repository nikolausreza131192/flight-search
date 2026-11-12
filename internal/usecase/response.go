package usecase

import "bookcabin/internal/entity"

// ScoredFlight is an entity flight enriched with the best value score
type ScoredFlight struct {
	entity.Flight
	BestValueScore float64 `json:"best_value_score"`
}

// Metadata describes how a result set was produced.
type Metadata struct {
	TotalResults       int   `json:"total_results"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}

// Itinerary is a round-trip pairing of a departure journey (origin →
// destination) and a return journey (destination → origin).
type Itinerary struct {
	ID               string          `json:"id"`
	TripType         string          `json:"trip_type"`
	DepartureJourney entity.Flight   `json:"departure_journey"`
	ReturnJourney    entity.Flight   `json:"return_journey"`
	TotalPrice       entity.Price    `json:"total_price"`
	TotalDuration    entity.Duration `json:"total_duration"`
	Stops            int             `json:"stops"`
	BestValueScore   float64         `json:"best_value_score"`
}

// LegResult holds the results of a single leg of a multi-city search.
type LegResult struct {
	SearchCriteria entity.SearchCriteria `json:"search_criteria"`
	Metadata       Metadata              `json:"metadata"`
	Flights        []ScoredFlight        `json:"flights"`
}

// SearchResponse is the top level API response.
type SearchResponse struct {
	SearchCriteria entity.SearchCriteria `json:"search_criteria"`
	Metadata       Metadata              `json:"metadata"`
	Flights        []ScoredFlight        `json:"flights"`
	Itineraries    []Itinerary           `json:"itineraries,omitempty"`
	Legs           []LegResult           `json:"legs,omitempty"`
}
