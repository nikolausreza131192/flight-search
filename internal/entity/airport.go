package entity

import (
	"strings"
	"time"
)

// Indonesian timezone identifiers (WIB / WITA / WIT).
const (
	TimezoneWIB  = "Asia/Jakarta"  // UTC+7
	TimezoneWITA = "Asia/Makassar" // UTC+8
	TimezoneWIT  = "Asia/Jayapura" // UTC+9
)

// Airport describes a single airport and the timezone it belongs to.
type Airport struct {
	Code     string
	City     string
	Name     string
	Timezone string
}

var airports = map[string]Airport{
	"CGK": {Code: "CGK", City: "Jakarta", Name: "Soekarno-Hatta International", Timezone: TimezoneWIB},
	"HLP": {Code: "HLP", City: "Jakarta", Name: "Halim Perdanakusuma", Timezone: TimezoneWIB},
	"DPS": {Code: "DPS", City: "Denpasar", Name: "Ngurah Rai International", Timezone: TimezoneWITA},
	"SUB": {Code: "SUB", City: "Surabaya", Name: "Juanda International", Timezone: TimezoneWIB},
	"SOC": {Code: "SOC", City: "Solo", Name: "Adi Soemarmo International", Timezone: TimezoneWIB},
	"UPG": {Code: "UPG", City: "Makassar", Name: "Sultan Hasanuddin International", Timezone: TimezoneWITA},
	"JOG": {Code: "JOG", City: "Yogyakarta", Name: "Yogyakarta International", Timezone: TimezoneWIB},
	"KNO": {Code: "KNO", City: "Medan", Name: "Kualanamu International", Timezone: TimezoneWIB},
	"DJJ": {Code: "DJJ", City: "Jayapura", Name: "Sentani", Timezone: TimezoneWIT},
	"DIL": {Code: "DIL", City: "Dili", Name: "Presidente Nicolau Lobato", Timezone: TimezoneWITA},
}

// LookupAirport returns the airport for a code (case-insensitive).
func LookupAirport(code string) (Airport, bool) {
	a, ok := airports[strings.ToUpper(strings.TrimSpace(code))]
	return a, ok
}

// CityFor returns the city for an airport code. A non-empty city supplied by the
// provider is preferred; otherwise the registry value is used, falling back to
// the code itself when it is unknown.
func CityFor(code, provided string) string {
	if strings.TrimSpace(provided) != "" {
		return provided
	}
	if a, ok := LookupAirport(code); ok {
		return a.City
	}
	return code
}

// LocationFor resolves the *time.Location for an airport code. It falls back to
// UTC when the code is unknown or the timezone database cannot be loaded.
func LocationFor(code string) *time.Location {
	if a, ok := LookupAirport(code); ok {
		if loc, err := time.LoadLocation(a.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}
