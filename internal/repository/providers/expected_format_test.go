package providers

import (
	"testing"

	"bookcabin/config"
)

// TestNormalizeAirAsiaMatchesExpected locks the normalized output of the AirAsia
// mock payload to the unified format shown in expected_result.json. Timestamps
// are intentionally excluded: the sample file carries placeholder Unix values
// that do not correspond to its own RFC3339 datetimes.
func TestNormalizeAirAsiaMatchesExpected(t *testing.T) {
	flights, err := NewAirAsia(config.Config{}).normalize(readFixture(t, "airasia_search_response.json"))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	var found bool
	for _, f := range flights {
		if f.FlightNumber != "QZ7250" {
			continue
		}
		found = true

		if f.ID != "QZ7250_AirAsia" {
			t.Errorf("id = %q, want %q", f.ID, "QZ7250_AirAsia")
		}
		if f.Provider != "AirAsia" {
			t.Errorf("provider = %q, want AirAsia", f.Provider)
		}
		if f.Airline.Name != "AirAsia" || f.Airline.Code != "QZ" {
			t.Errorf("airline = %+v, want {AirAsia QZ}", f.Airline)
		}
		if f.Departure.Airport != "CGK" || f.Departure.City != "Jakarta" {
			t.Errorf("departure = %+v", f.Departure)
		}
		if f.Departure.Datetime != "2025-12-15T15:15:00+07:00" {
			t.Errorf("departure datetime = %q", f.Departure.Datetime)
		}
		if f.Arrival.Airport != "DPS" || f.Arrival.City != "Denpasar" {
			t.Errorf("arrival = %+v", f.Arrival)
		}
		if f.Arrival.Datetime != "2025-12-15T20:35:00+08:00" {
			t.Errorf("arrival datetime = %q", f.Arrival.Datetime)
		}
		if f.Duration.TotalMinutes != 260 || f.Duration.Formatted != "4h 20m" {
			t.Errorf("duration = %+v, want {260 4h 20m}", f.Duration)
		}
		if f.Stops != 1 {
			t.Errorf("stops = %d, want 1", f.Stops)
		}
		if f.Price.Amount != 485000 || f.Price.Currency != "IDR" {
			t.Errorf("price = %+v, want {485000 IDR}", f.Price)
		}
		if f.AvailableSeats != 88 {
			t.Errorf("available_seats = %d, want 88", f.AvailableSeats)
		}
		if f.CabinClass != "economy" {
			t.Errorf("cabin_class = %q, want economy", f.CabinClass)
		}
		if f.Aircraft != nil {
			t.Errorf("aircraft = %v, want nil", *f.Aircraft)
		}
		if f.Amenities == nil || len(f.Amenities) != 0 {
			t.Errorf("amenities = %v, want empty slice", f.Amenities)
		}
		if f.Baggage.CarryOn != "Cabin baggage only" || f.Baggage.Checked != "Additional fee" {
			t.Errorf("baggage = %+v", f.Baggage)
		}
	}
	if !found {
		t.Fatal("QZ7250 not found in normalized output")
	}
}

// TestNormalizeGarudaConnectingFlight verifies the multi-segment inconsistency
// (top level arrival points at the first stop, segments reveal the real
// destination) is resolved.
func TestNormalizeGarudaConnectingFlight(t *testing.T) {
	flights, err := NewGaruda(config.Config{}).normalize(readFixture(t, "garuda_indonesia_search_response.json"))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	for _, f := range flights {
		if f.FlightNumber != "GA315" {
			continue
		}
		if f.Arrival.Airport != "DPS" {
			t.Errorf("arrival airport = %q, want DPS (resolved from segments)", f.Arrival.Airport)
		}
		if f.Stops != 1 {
			t.Errorf("stops = %d, want 1", f.Stops)
		}
		if len(f.Segments) != 2 {
			t.Errorf("segments = %d, want 2", len(f.Segments))
		}
		return
	}
	t.Fatal("GA315 not found")
}
