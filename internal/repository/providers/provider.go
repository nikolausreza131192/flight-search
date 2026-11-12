package providers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"bookcabin/config"
	"bookcabin/internal/util/ratelimit"
)

// errUpstreamUnavailable emulates a transient failure of the live upstream.
var errUpstreamUnavailable = errors.New("upstream service temporarily unavailable")

// baseProvider carries the behaviour every airline shares
type baseProvider struct {
	code       string
	airlineCfg config.Airline
	limiter    *ratelimit.Limiter
	simulate   bool
}

// newBaseProvider resolves one airline's settings from config and builds its limiter.
func newBaseProvider(code string, cfg config.Config) baseProvider {
	airlineCfg := cfg.Airlines[code]
	return baseProvider{
		code:       code,
		airlineCfg: airlineCfg,
		limiter:    ratelimit.New(airlineCfg.Search.RateLimit, airlineCfg.Search.RateLimitWindow),
		simulate:   cfg.SimulateLatency,
	}
}

// Code returns the airline's IATA code.
func (b baseProvider) Code() string { return b.code }

// read returns the airline's raw response body
func (b baseProvider) read(ctx context.Context) ([]byte, error) {
	if err := b.limiter.Allow(); err != nil {
		return nil, err
	}
	if b.simulate {
		if err := delay(ctx, b.airlineCfg.Search.MinDelay, b.airlineCfg.Search.MaxDelay); err != nil {
			return nil, err
		}
	}

	if b.airlineCfg.Search.FailureRate > 0 && rand.Float64() < b.airlineCfg.Search.FailureRate {
		return nil, errUpstreamUnavailable
	}

	data, err := os.ReadFile(b.airlineCfg.Search.File)
	if err != nil {
		return nil, fmt.Errorf("read fixture %s: %w", b.airlineCfg.Search.File, err)
	}
	return data, nil
}

// delay waits for a random duration in [min, max], returning early if the
// context is cancelled.
func delay(ctx context.Context, min, max time.Duration) error {
	delay := min
	if max > min {
		delay += time.Duration(rand.Int63n(int64(max - min)))
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
