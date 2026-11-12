package repository

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"bookcabin/config"
	"bookcabin/internal/entity"
)

// fakeProvider is a configurable AirlineProvider used to exercise the
// SearchRepo.
type fakeProvider struct {
	code      string
	calls     int32
	failFirst int
	alwaysErr bool
	delay     time.Duration
	flights   []entity.Flight
}

func (f *fakeProvider) Code() string { return f.code }

func (f *fakeProvider) Search(ctx context.Context, _ entity.SearchCriteria) ([]entity.Flight, error) {
	n := atomic.AddInt32(&f.calls, 1)
	if f.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(f.delay):
		}
	}
	if f.alwaysErr || int(n) <= f.failFirst {
		return nil, errors.New("boom")
	}
	return f.flights, nil
}

func testConfig() config.Config {
	return config.Config{
		SearchTimeout:  time.Second,
		MaxRetries:     2,
		RetryBaseDelay: time.Millisecond,
	}
}

func TestSearchRepoRetriesTransientFailure(t *testing.T) {
	p := &fakeProvider{
		code:      "JT",
		failFirst: 1,
		flights:   []entity.Flight{{FlightNumber: "X1"}},
	}
	repo := NewSearchRepo([]AirlineProvider{p}, testConfig())

	res := repo.Search(context.Background(), entity.SearchCriteria{})
	if res.Succeeded != 1 || res.Failed != 0 {
		t.Fatalf("expected success after retry, got %+v", res)
	}
	if res.Queried != 1 || len(res.Flights) != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if got := atomic.LoadInt32(&p.calls); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestSearchRepoReportsPermanentFailure(t *testing.T) {
	p := &fakeProvider{code: "down", alwaysErr: true}
	repo := NewSearchRepo([]AirlineProvider{p}, testConfig())

	res := repo.Search(context.Background(), entity.SearchCriteria{})
	if res.Failed != 1 || res.Succeeded != 0 {
		t.Fatalf("expected 1 failure, got %+v", res)
	}
	if _, ok := res.Errors["down"]; !ok {
		t.Fatalf("expected error recorded for provider, got %+v", res.Errors)
	}
}

func TestSearchRepoHonoursTimeout(t *testing.T) {
	slow := &fakeProvider{code: "slow", delay: 5 * time.Second, flights: []entity.Flight{{FlightNumber: "X"}}}
	cfg := testConfig()
	cfg.SearchTimeout = 50 * time.Millisecond
	repo := NewSearchRepo([]AirlineProvider{slow}, cfg)

	start := time.Now()
	res := repo.Search(context.Background(), entity.SearchCriteria{})
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("search did not respect timeout, took %s", elapsed)
	}
	if res.Failed != 1 {
		t.Fatalf("expected timeout to register as failure, got %+v", res)
	}
}
