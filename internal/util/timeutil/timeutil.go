// Package timeutil centralizes date/time parsing and formatting so that the
// wildly different timestamp formats returned by providers can be normalized
// into a single RFC3339 + Unix timestamp representation.
package timeutil

import (
	"fmt"
	"strings"
	"time"
)

// DateLayout is the layout used for the "departureDate" query field.
const DateLayout = "2006-01-02"

// parseLayouts lists every layout we try, from most to least specific.
var parseLayouts = []string{
	time.RFC3339Nano,           // 2025-12-15T06:00:00+07:00
	time.RFC3339,               // 2025-12-15T06:00:00+07:00
	"2006-01-02T15:04:05-0700", // 2025-12-15T07:15:00+0700 (Batik Air)
	"2006-01-02T15:04:05Z0700", // 2025-12-15T07:15:00+0700
	"2006-01-02T15:04:05",      // naive (Lion Air, timezone supplied separately)
	"2006-01-02 15:04:05",      // naive with space
	"2006-01-02T15:04",         // without seconds
}

// Parse attempts to parse a timestamp using every supported layout.
// use ParseInLocation when the provider reports a timezone separately.
func Parse(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range parseLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", value)
}

// ParseInLocation parses a naive local timestamp and attaches the given IANA
// timezone name (e.g. "Asia/Jakarta"). If the timestamp already carries an
// offset it is parsed as-is.
func ParseInLocation(value, timezone string) (time.Time, error) {
	v := strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05-0700"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return time.Time{}, fmt.Errorf("unknown timezone %q: %w", timezone, err)
	}
	naive, err := time.ParseInLocation("2006-01-02T15:04:05", v, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognized timestamp %q", value)
	}
	return naive, nil
}

// Format renders a time as RFC3339
func Format(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatDuration converts a number of minutes into "4h 20m" style output.
func FormatDuration(minutes int) string {
	if minutes <= 0 {
		return "0m"
	}
	h := minutes / 60
	m := minutes % 60
	switch {
	case h == 0:
		return fmt.Sprintf("%dm", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh %dm", h, m)
	}
}

// AtTimeOnDate rebuilds a timestamp on the calendar date of base while keeping
// the clock time and location of t.
func AtTimeOnDate(base, t time.Time) time.Time {
	return time.Date(base.Year(), base.Month(), base.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// MinutesOfDay parses an "HH:MM" string into minutes since midnight.
func MinutesOfDay(value string) (int, error) {
	t, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid time %q, expected HH:MM", value)
	}
	return t.Hour()*60 + t.Minute(), nil
}
