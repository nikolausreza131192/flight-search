package repository

import (
	"bookcabin/config"
	"bookcabin/internal/entity"
	"bookcabin/internal/repository/providers"
	"context"
)

// AirlineProvider is the outbound port implemented by each airline adapter.
type AirlineProvider interface {
	// Code returns the unique provider code (e.g. "GI").
	Code() string
	// Search returns the normalized flights for the given criteria.
	Search(ctx context.Context, criteria entity.SearchCriteria) ([]entity.Flight, error)
}

// NewProviders builds one adapter per airline. Each adapter owns its own
// response contract, rate limiter and latency profile — all resolved from
// config.Airlines by IATA code — so registering a new airline is a matter of
// adding its constructor here.
func NewProviders(cfg config.Config) []AirlineProvider {
	return []AirlineProvider{
		providers.NewGaruda(cfg),
		providers.NewLion(cfg),
		providers.NewBatik(cfg),
		providers.NewAirAsia(cfg),
	}
}
