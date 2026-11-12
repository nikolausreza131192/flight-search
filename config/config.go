// Package config loads runtime configuration from a JSON file kept in the
// template directory, falling back to sensible defaults suitable for local
// development.
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Config holds every tunable of the service.
type Config struct {
	Addr            string
	LogLevel        slog.Level
	CacheTTL        time.Duration
	SearchTimeout   time.Duration
	MaxRetries      int
	RetryBaseDelay  time.Duration
	SimulateLatency bool

	// Airlines holds the per-airline settings, keyed by IATA airline code
	// (e.g. "GI" for Garuda Indonesia, "QZ" for AirAsia).
	Airlines map[string]Airline
}

// Airline groups the settings of a single upstream airline.
type Airline struct {
	Search AirlineSearch
}

// AirlineSearch tunes one airline's search upstream
type AirlineSearch struct {
	// File is the canned response file, resolved relative to the directory
	// holding the configuration file.
	File            string
	RateLimit       int
	RateLimitWindow time.Duration
	MinDelay        time.Duration
	MaxDelay        time.Duration
	FailureRate     float64
}

// Load reads the JSON configuration file at path and converts it into a Config.
// Airline fixture paths are resolved relative to the file's directory, so they
// work regardless of the working directory the process starts in.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return f.toConfig(filepath.Dir(path))
}

// file mirrors the JSON on disk. Durations are plain strings (e.g. "5m") and the
// log level its textual form, neither of which json.Unmarshal can decode into
// the runtime time.Duration / slog.Level fields directly.
type file struct {
	Addr            string                  `json:"addr"`
	LogLevel        string                  `json:"log_level"`
	CacheTTL        string                  `json:"cache_ttl"`
	SearchTimeout   string                  `json:"search_timeout"`
	MaxRetries      int                     `json:"max_retries"`
	RetryBaseDelay  string                  `json:"retry_base_delay"`
	SimulateLatency bool                    `json:"simulate_latency"`
	Airlines        map[string]airlineEntry `json:"airlines"`
}

type airlineEntry struct {
	Search airlineSearchEntry `json:"search"`
}

type airlineSearchEntry struct {
	File            string  `json:"file"`
	RateLimit       int     `json:"rate_limit"`
	RateLimitWindow string  `json:"rate_limit_window"`
	MinDelay        string  `json:"min_delay"`
	MaxDelay        string  `json:"max_delay"`
	FailureRate     float64 `json:"failure_rate"`
}

// toConfig validates the decoded file and converts it into a Config.
func (f file) toConfig(dir string) (Config, error) {
	level, err := parseLevel(f.LogLevel)
	if err != nil {
		return Config{}, err
	}
	cacheTTL, err := parseDuration("cache_ttl", f.CacheTTL)
	if err != nil {
		return Config{}, err
	}
	searchTimeout, err := parseDuration("search_timeout", f.SearchTimeout)
	if err != nil {
		return Config{}, err
	}
	retryBaseDelay, err := parseDuration("retry_base_delay", f.RetryBaseDelay)
	if err != nil {
		return Config{}, err
	}

	airlines := make(map[string]Airline, len(f.Airlines))
	for code, a := range f.Airlines {
		window, err := parseDuration("airlines."+code+".search.rate_limit_window", a.Search.RateLimitWindow)
		if err != nil {
			return Config{}, err
		}
		minDelay, err := parseDuration("airlines."+code+".search.min_delay", a.Search.MinDelay)
		if err != nil {
			return Config{}, err
		}
		maxDelay, err := parseDuration("airlines."+code+".search.max_delay", a.Search.MaxDelay)
		if err != nil {
			return Config{}, err
		}
		airlines[code] = Airline{Search: AirlineSearch{
			File:            filepath.Join(dir, a.Search.File),
			RateLimit:       a.Search.RateLimit,
			RateLimitWindow: window,
			MinDelay:        minDelay,
			MaxDelay:        maxDelay,
			FailureRate:     a.Search.FailureRate,
		}}
	}

	return Config{
		Addr:            f.Addr,
		LogLevel:        level,
		CacheTTL:        cacheTTL,
		SearchTimeout:   searchTimeout,
		MaxRetries:      f.MaxRetries,
		RetryBaseDelay:  retryBaseDelay,
		SimulateLatency: f.SimulateLatency,
		Airlines:        airlines,
	}, nil
}

func parseDuration(field, value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", field, value, err)
	}
	return d, nil
}

func parseLevel(value string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("invalid log_level %q: %w", value, err)
	}
	return level, nil
}
