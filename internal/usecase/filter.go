package usecase

import (
	"sort"
	"strings"

	"bookcabin/internal/entity"
	"bookcabin/internal/util/timeutil"
)

// filter returns the subset of flights matching every provided filter. A nil
// filter returns the input unchanged.
func filter(flights []entity.Flight, filter *Filters) []entity.Flight {
	if filter == nil || isEmpty(filter) {
		return flights
	}

	out := make([]entity.Flight, 0, len(flights))
	for _, f := range flights {
		if matches(f, filter) {
			out = append(out, f)
		}
	}
	return out
}

func isEmpty(f *Filters) bool {
	return f.MinPrice == nil && f.MaxPrice == nil && f.MaxStops == nil &&
		len(f.Airlines) == 0 && f.DepartureTimeFrom == nil && f.DepartureTimeTo == nil &&
		f.ArrivalTimeFrom == nil && f.ArrivalTimeTo == nil && f.MaxDurationMinutes == nil
}

func matches(f entity.Flight, filter *Filters) bool {
	price := float64(f.Price.Amount)
	if filter.MinPrice != nil && price < *filter.MinPrice {
		return false
	}
	if filter.MaxPrice != nil && price > *filter.MaxPrice {
		return false
	}
	if filter.MaxStops != nil && f.Stops > *filter.MaxStops {
		return false
	}
	if filter.MaxDurationMinutes != nil && f.Duration.TotalMinutes > *filter.MaxDurationMinutes {
		return false
	}
	if len(filter.Airlines) > 0 && !airlineMatches(f, filter.Airlines) {
		return false
	}
	if !timeInRange(f.Departure.Datetime, filter.DepartureTimeFrom, filter.DepartureTimeTo) {
		return false
	}
	if !timeInRange(f.Arrival.Datetime, filter.ArrivalTimeFrom, filter.ArrivalTimeTo) {
		return false
	}
	return true
}

func airlineMatches(f entity.Flight, airlines []string) bool {
	for _, a := range airlines {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if strings.EqualFold(a, f.Airline.Code) {
			return true
		}
	}
	return false
}

// timeInRange checks the local time-of-day of an RFC3339 timestamp against an
// optional [from, to] window expressed as "HH:MM".
func timeInRange(timestamp string, from, to *string) bool {
	if from == nil && to == nil {
		return true
	}
	t, err := timeutil.Parse(timestamp)
	if err != nil {
		return true
	}
	current := t.Hour()*60 + t.Minute()
	if from != nil {
		if v, err := timeutil.MinutesOfDay(*from); err == nil && current < v {
			return false
		}
	}
	if to != nil {
		if v, err := timeutil.MinutesOfDay(*to); err == nil && current > v {
			return false
		}
	}
	return true
}

// Sort orders the flights in place according to the sort spec. When the spec is
// nil it defaults to best value, highest first.
func Sort(flights []ScoredFlight, spec *SortSpec) {
	by, order := SortBestValue, ""
	if spec != nil {
		if spec.By != "" {
			by = spec.By
		}
		order = spec.Order
	}
	if order == "" {
		order = defaultOrder(by)
	}

	less := comparator(by)
	sort.SliceStable(flights, func(i, j int) bool {
		if order == "desc" {
			return less(flights[j], flights[i])
		}
		return less(flights[i], flights[j])
	})
}

// defaultOrder returns the natural ordering for a sort key.
func defaultOrder(by string) string {
	switch by {
	case SortPrice, SortDuration, SortDepartureTime, SortArrivalTime:
		return "asc"
	default:
		return "desc"
	}
}

// comparator returns a "less" function for the given sort key.
func comparator(by string) func(a, b ScoredFlight) bool {
	switch by {
	case SortPrice:
		return func(a, b ScoredFlight) bool { return a.Price.Amount < b.Price.Amount }
	case SortDuration:
		return func(a, b ScoredFlight) bool { return a.Duration.TotalMinutes < b.Duration.TotalMinutes }
	case SortDepartureTime:
		return func(a, b ScoredFlight) bool { return a.Departure.Timestamp < b.Departure.Timestamp }
	case SortArrivalTime:
		return func(a, b ScoredFlight) bool { return a.Arrival.Timestamp < b.Arrival.Timestamp }
	default: // best_value
		return func(a, b ScoredFlight) bool { return a.BestValueScore < b.BestValueScore }
	}
}
