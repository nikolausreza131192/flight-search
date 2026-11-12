package usecase

import (
	"math"

	"bookcabin/internal/entity"
)

// Weights used to compose the best value score (based on price + convenience)
const (
	WeightPrice    = 0.5
	WeightDuration = 0.3
	WeightStops    = 0.2
)

// applyScore wraps the flights with a BestValueScore
func applyScore(flights []entity.Flight) []ScoredFlight {
	if len(flights) == 0 {
		return nil
	}

	scored := make([]ScoredFlight, len(flights))
	for i, f := range flights {
		scored[i] = ScoredFlight{Flight: f}
	}

	minPrice, maxPrice := priceBounds(scored)
	minDur, maxDur := durationBounds(scored)
	stopsCeiling := maxStops(scored)

	for i := range scored {
		f := &scored[i]
		price := inverseNormalize(float64(f.Price.Amount), minPrice, maxPrice)
		duration := inverseNormalize(float64(f.Duration.TotalMinutes), minDur, maxDur)
		stops := 1.0
		if stopsCeiling > 0 {
			stops = 1 - float64(f.Stops)/float64(stopsCeiling)
		}
		f.BestValueScore = round(100*(WeightPrice*price+WeightDuration*duration+WeightStops*stops), 2)
	}
	return scored
}

// inverseNormalize maps v into [0,1] where a lower input yields a higher output.
func inverseNormalize(v, min, max float64) float64 {
	if max <= min {
		return 1
	}
	return (max - v) / (max - min)
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

func priceBounds(flights []ScoredFlight) (float64, float64) {
	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, f := range flights {
		v := float64(f.Price.Amount)
		min = math.Min(min, v)
		max = math.Max(max, v)
	}
	return min, max
}

func durationBounds(flights []ScoredFlight) (float64, float64) {
	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, f := range flights {
		v := float64(f.Duration.TotalMinutes)
		min = math.Min(min, v)
		max = math.Max(max, v)
	}
	return min, max
}

func maxStops(flights []ScoredFlight) int {
	max := 0
	for _, f := range flights {
		if f.Stops > max {
			max = f.Stops
		}
	}
	return max
}
