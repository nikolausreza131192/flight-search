package usecase

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"bookcabin/internal/entity"
	"bookcabin/internal/util/currency"
	"bookcabin/internal/util/timeutil"
)

// maxItineraryCombinations caps the round-trip pairing work.
const maxItineraryCombinations = 100

// Options tunes the search use case.
type Options struct {
	// CacheTTL is how long a response may be reused. A value <= 0 disables caching.
	CacheTTL time.Duration
}

// SearchUseCase is the default SearchItf implementation.
type SearchUseCase struct {
	cache    ResultCache
	repo     SearchRepository
	cacheTTL time.Duration
}

// New builds a search SearchUseCase with its collaborators injected as ports.
func New(cache ResultCache, repo SearchRepository, opts Options) *SearchUseCase {
	return &SearchUseCase{cache: cache, repo: repo, cacheTTL: opts.CacheTTL}
}

// Search validates the request, serves it from cache when possible and
// otherwise runs the appropriate search strategy.
func (s *SearchUseCase) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	start := time.Now()
	normalize(&req)
	if err := validate(req); err != nil {
		return SearchResponse{}, err
	}

	key, keyErr := cacheKey(req)
	cacheable := keyErr == nil && s.cacheTTL > 0
	if cacheable {
		if cached, ok := s.cache.Get(key); ok {
			cached.Metadata.CacheHit = true
			// The stored entry carries the duration of the search that filled the
			// cache; report this request's own (cache-hit) duration instead.
			cached.Metadata.SearchTimeMs = time.Since(start).Milliseconds()
			return cached, nil
		}
	}

	var resp SearchResponse
	switch {
	case len(req.MultiCity) > 0:
		resp = s.searchMultiCity(ctx, req)
	case req.ReturnDate != nil && *req.ReturnDate != "":
		resp = s.searchRoundTrip(ctx, req)
	default:
		resp = s.searchOneWay(ctx, req)
	}
	resp.Metadata.SearchTimeMs = time.Since(start).Milliseconds()

	if cacheable {
		s.cache.Set(key, resp)
	}
	return resp, nil
}

// searchOneWay runs a single origin/destination search.
func (s *SearchUseCase) searchOneWay(ctx context.Context, req SearchRequest) SearchResponse {
	criteria, meta, flights := s.runLeg(ctx, req)
	return SearchResponse{SearchCriteria: criteria, Metadata: meta, Flights: flights}
}

// searchRoundTrip searches both the departure and the return journey and pairs
// the best combinations.
func (s *SearchUseCase) searchRoundTrip(ctx context.Context, req SearchRequest) SearchResponse {
	departureReq := req
	departureReq.ReturnDate = nil
	departureReq.Filters = nil
	departureReq.Sort = nil

	returnReq := req
	returnReq.Origin, returnReq.Destination = req.Destination, req.Origin
	returnReq.DepartureDate = *req.ReturnDate
	returnReq.ReturnDate = nil
	returnReq.Filters = nil
	returnReq.Sort = nil

	_, outMeta, departureJourney := s.runLeg(ctx, departureReq)
	_, _, returnJourney := s.runLeg(ctx, returnReq)

	itineraries := buildItineraries(departureJourney, returnJourney)
	return SearchResponse{
		SearchCriteria: toCriteria(req),
		Metadata: Metadata{
			TotalResults:       len(itineraries),
			ProvidersQueried:   outMeta.ProvidersQueried,
			ProvidersSucceeded: outMeta.ProvidersSucceeded,
			ProvidersFailed:    outMeta.ProvidersFailed,
		},
		Flights:     departureJourney,
		Itineraries: itineraries,
	}
}

// searchMultiCity searches every leg independently.
func (s *SearchUseCase) searchMultiCity(ctx context.Context, req SearchRequest) SearchResponse {
	legs := make([]LegResult, 0, len(req.MultiCity))
	var total, queried, succeeded, failed int

	for _, leg := range req.MultiCity {
		legReq := SearchRequest{
			Origin:        leg.Origin,
			Destination:   leg.Destination,
			DepartureDate: leg.DepartureDate,
			Passengers:    req.Passengers,
			CabinClass:    req.CabinClass,
			Filters:       req.Filters,
			Sort:          req.Sort,
		}
		criteria, meta, flights := s.runLeg(ctx, legReq)
		legs = append(legs, LegResult{SearchCriteria: criteria, Metadata: meta, Flights: flights})
		total += meta.TotalResults
		queried, succeeded, failed = meta.ProvidersQueried, meta.ProvidersSucceeded, meta.ProvidersFailed
	}

	return SearchResponse{
		SearchCriteria: toCriteria(req),
		Metadata: Metadata{
			TotalResults:       total,
			ProvidersQueried:   queried,
			ProvidersSucceeded: succeeded,
			ProvidersFailed:    failed,
		},
		Flights: []ScoredFlight{},
		Legs:    legs,
	}
}

// runLeg performs one origin/destination search end-to-end.
func (s *SearchUseCase) runLeg(ctx context.Context, req SearchRequest) (entity.SearchCriteria, Metadata, []ScoredFlight) {
	start := time.Now()
	criteria := toCriteria(req)

	res := s.repo.Search(ctx, criteria)

	applyPassengers(res.Flights, req.Passengers)
	flights := applyScore(filter(res.Flights, req.Filters))
	Sort(flights, req.Sort)
	if flights == nil {
		flights = []ScoredFlight{}
	}

	meta := Metadata{
		TotalResults:       len(flights),
		ProvidersQueried:   res.Queried,
		ProvidersSucceeded: res.Succeeded,
		ProvidersFailed:    res.Failed,
		SearchTimeMs:       time.Since(start).Milliseconds(),
	}
	return criteria, meta, flights
}

// buildItineraries pairs departure and return journeys, ranked by best value.
func buildItineraries(departureJourney, returnJourney []ScoredFlight) []Itinerary {
	if len(departureJourney) == 0 || len(returnJourney) == 0 {
		return nil
	}

	itineraries := make([]Itinerary, 0, len(departureJourney)*len(returnJourney))
	for _, departure := range departureJourney {
		for _, ret := range returnJourney {
			amount := departure.Price.Amount + ret.Price.Amount
			totalMinutes := departure.Duration.TotalMinutes + ret.Duration.TotalMinutes
			itineraries = append(itineraries, Itinerary{
				ID:               departure.ID + "::" + ret.ID,
				TripType:         "round_trip",
				DepartureJourney: departure.Flight,
				ReturnJourney:    ret.Flight,
				TotalPrice: entity.Price{
					Amount:    amount,
					Currency:  departure.Price.Currency,
					Formatted: currency.Format(amount, departure.Price.Currency),
				},
				TotalDuration: entity.Duration{
					TotalMinutes: totalMinutes,
					Formatted:    timeutil.FormatDuration(totalMinutes),
				},
				Stops: departure.Stops + ret.Stops,
			})
			if len(itineraries) >= maxItineraryCombinations {
				break
			}
		}
		if len(itineraries) >= maxItineraryCombinations {
			break
		}
	}

	scoreItineraries(itineraries)
	sort.SliceStable(itineraries, func(i, j int) bool {
		return itineraries[i].BestValueScore > itineraries[j].BestValueScore
	})
	return itineraries
}

// scoreItineraries applies the same best-value weighting at itinerary level.
func scoreItineraries(itineraries []Itinerary) {
	if len(itineraries) == 0 {
		return
	}
	proxies := make([]entity.Flight, len(itineraries))
	for i, it := range itineraries {
		proxies[i] = entity.Flight{
			Price:    it.TotalPrice,
			Duration: it.TotalDuration,
			Stops:    it.Stops,
		}
	}
	scored := applyScore(proxies)
	for i := range itineraries {
		itineraries[i].BestValueScore = scored[i].BestValueScore
	}
}

// applyPassengers scales fares to the total for the requested party size.
func applyPassengers(flights []entity.Flight, passengers int) {
	if passengers <= 1 {
		return
	}
	for i := range flights {
		flights[i].Price.Amount *= int64(passengers)
		flights[i].Price.Formatted = currency.Format(flights[i].Price.Amount, flights[i].Price.Currency)
	}
}

// normalize fills in request defaults.
func normalize(req *SearchRequest) {
	req.Origin = strings.ToUpper(strings.TrimSpace(req.Origin))
	req.Destination = strings.ToUpper(strings.TrimSpace(req.Destination))
	if req.Passengers <= 0 {
		req.Passengers = 1
	}
	if strings.TrimSpace(req.CabinClass) == "" {
		req.CabinClass = "economy"
	}
	for i := range req.MultiCity {
		req.MultiCity[i].Origin = strings.ToUpper(strings.TrimSpace(req.MultiCity[i].Origin))
		req.MultiCity[i].Destination = strings.ToUpper(strings.TrimSpace(req.MultiCity[i].Destination))
	}
}

// validate checks the request is well formed and searchable.
func validate(req SearchRequest) error {
	if len(req.MultiCity) > 0 {
		for i, leg := range req.MultiCity {
			if leg.Origin == "" || leg.Destination == "" {
				return entity.NewValidationError("multiCity[%d]: origin and destination are required", i)
			}
			if err := validateDate(leg.DepartureDate); err != nil {
				return entity.NewValidationError("multiCity[%d]: %s", i, err)
			}
		}
		return nil
	}

	if req.Origin == "" {
		return entity.NewValidationError("origin is required")
	}
	if req.Destination == "" {
		return entity.NewValidationError("destination is required")
	}
	if req.Origin == req.Destination {
		return entity.NewValidationError("origin and destination must differ")
	}
	if err := validateDate(req.DepartureDate); err != nil {
		return err
	}
	if req.ReturnDate != nil && *req.ReturnDate != "" {
		if err := validateDate(*req.ReturnDate); err != nil {
			return entity.NewValidationError("returnDate: %s", err)
		}
		if *req.ReturnDate < req.DepartureDate {
			return entity.NewValidationError("returnDate must not be before departureDate")
		}
	}
	return nil
}

func validateDate(value string) error {
	if strings.TrimSpace(value) == "" {
		return entity.NewValidationError("departureDate is required")
	}
	if _, err := time.Parse(timeutil.DateLayout, value); err != nil {
		return entity.NewValidationError("departureDate must use YYYY-MM-DD format")
	}
	return nil
}

// toCriteria maps a request to the normalized search criteria value object.
func toCriteria(req SearchRequest) entity.SearchCriteria {
	criteria := entity.SearchCriteria{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
	}
	if req.ReturnDate != nil {
		criteria.ReturnDate = *req.ReturnDate
	}
	for _, leg := range req.MultiCity {
		criteria.MultiCity = append(criteria.MultiCity, entity.MultiCityLeg{
			Origin:        leg.Origin,
			Destination:   leg.Destination,
			DepartureDate: leg.DepartureDate,
		})
	}
	return criteria
}

// cacheKey derives a stable cache key from the searchable parts of a request.
func cacheKey(req SearchRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
