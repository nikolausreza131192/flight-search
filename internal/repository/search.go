package repository

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"bookcabin/config"
	"bookcabin/internal/entity"
	"bookcabin/internal/usecase"
)

// SearchRepo implements the usecase.SearchRepository interface
type SearchRepo struct {
	providers []AirlineProvider
	cfg       config.Config
}

// NewSearchRepo creates a SearchRepo over the given providers.
func NewSearchRepo(providers []AirlineProvider, cfg config.Config) *SearchRepo {
	return &SearchRepo{providers: providers, cfg: cfg}
}

// Search queries every provider in parallel and returns the merged result.
func (r *SearchRepo) Search(ctx context.Context, criteria entity.SearchCriteria) usecase.SearchResult {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.SearchTimeout)
	defer cancel()

	type providerResult struct {
		code    string
		flights []entity.Flight
		err     error
	}

	ch := make(chan providerResult, len(r.providers))
	for _, p := range r.providers {
		go func(p AirlineProvider) {
			flights, err := r.fetchWithRetry(ctx, p, criteria)
			ch <- providerResult{code: p.Code(), flights: flights, err: err}
		}(p)
	}

	result := usecase.SearchResult{Queried: len(r.providers), Errors: make(map[string]string)}
	for range r.providers {
		o := <-ch
		if o.err != nil {
			result.Failed++
			result.Errors[o.code] = o.err.Error()
			slog.Warn("provider query failed", "provider", o.code, "error", o.err)
			continue
		}
		result.Succeeded++
		result.Flights = append(result.Flights, o.flights...)
	}

	return result
}

// fetchWithRetry calls a provider, retrying transient failures with exponential
// backoff (plus jitter) until the context deadline or the max retries.
func (r *SearchRepo) fetchWithRetry(ctx context.Context, p AirlineProvider, criteria entity.SearchCriteria) ([]entity.Flight, error) {
	var lastErr error
	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := backoff(ctx, r.cfg.RetryBaseDelay, attempt); err != nil {
				return nil, err
			}
		}
		flights, err := p.Search(ctx, criteria)
		if err == nil {
			return flights, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("%s failed after %d attempts: %w", p.Code(), r.cfg.MaxRetries+1, lastErr)
}

// backoff sleeps for base * 2^(attempt-1) plus up to 50% jitter.
func backoff(ctx context.Context, base time.Duration, attempt int) error {
	if base <= 0 {
		return nil
	}
	delay := base * time.Duration(1<<(attempt-1))
	delay += time.Duration(rand.Int63n(int64(delay/2) + 1))

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
