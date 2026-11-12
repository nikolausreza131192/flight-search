package usecase

import (
	"testing"

	"bookcabin/internal/entity"
)

func f(number string, code string, price int64, minutes, stops int) entity.Flight {
	return entity.Flight{
		FlightNumber: number,
		Airline:      entity.Airline{Name: code + " Air", Code: code},
		Price:        entity.Price{Amount: price},
		Duration:     entity.Duration{TotalMinutes: minutes},
		Stops:        stops,
		Departure:    entity.Endpoint{Datetime: "2025-12-15T06:00:00+07:00", Timestamp: 100},
		Arrival:      entity.Endpoint{Datetime: "2025-12-15T09:00:00+08:00", Timestamp: 200},
	}
}

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }
func ptrS(v string) *string   { return &v }

func TestApplyFilters(t *testing.T) {
	flights := []entity.Flight{
		f("A1", "QZ", 500000, 100, 0),
		f("A2", "JT", 900000, 200, 1),
		f("A3", "GA", 1500000, 150, 0),
	}

	got := filter(flights, &Filters{MaxStops: ptrI(0), MaxPrice: ptrF(1000000)})
	if len(got) != 1 || got[0].FlightNumber != "A1" {
		t.Fatalf("expected only A1, got %+v", got)
	}

	got = filter(flights, &Filters{Airlines: []string{"GA"}})
	if len(got) != 1 || got[0].FlightNumber != "A3" {
		t.Fatalf("expected only A3, got %+v", got)
	}

	got = filter(flights, &Filters{MaxDurationMinutes: ptrI(120)})
	if len(got) != 1 || got[0].FlightNumber != "A1" {
		t.Fatalf("expected only A1 by duration, got %+v", got)
	}

	if got := filter(flights, nil); len(got) != 3 {
		t.Fatalf("nil filter should return all, got %d", len(got))
	}
}

func TestApplyTimeOfDayFilter(t *testing.T) {
	flights := []entity.Flight{
		f("A1", "QZ", 500000, 100, 0), // departs 06:00
		f("A2", "JT", 900000, 200, 1), // departs 06:00
	}
	got := filter(flights, &Filters{DepartureTimeFrom: ptrS("05:00"), DepartureTimeTo: ptrS("05:30")})
	if len(got) != 0 {
		t.Fatalf("expected no flights in 05:00-05:30 window, got %d", len(got))
	}
	got = filter(flights, &Filters{DepartureTimeFrom: ptrS("05:00"), DepartureTimeTo: ptrS("07:00")})
	if len(got) != 2 {
		t.Fatalf("expected both flights in 05:00-07:00 window, got %d", len(got))
	}
}

func TestSortByPrice(t *testing.T) {
	flights := []ScoredFlight{
		{Flight: f("A1", "QZ", 500000, 100, 0)},
		{Flight: f("A2", "JT", 900000, 200, 1)},
		{Flight: f("A3", "GA", 300000, 150, 0)},
	}
	Sort(flights, &SortSpec{By: SortPrice, Order: "asc"})
	want := []string{"A3", "A1", "A2"}
	for i, w := range want {
		if flights[i].FlightNumber != w {
			t.Fatalf("price asc order = %v, want %v", numbers(flights), want)
		}
	}

	Sort(flights, &SortSpec{By: SortPrice, Order: "desc"})
	if flights[0].FlightNumber != "A2" {
		t.Fatalf("price desc first = %s, want A2", flights[0].FlightNumber)
	}
}

func TestSortDefaultIsBestValueDesc(t *testing.T) {
	flights := []ScoredFlight{
		{Flight: f("A1", "QZ", 500000, 100, 0)},
		{Flight: f("A2", "JT", 900000, 200, 1)},
	}
	flights[0].BestValueScore = 10
	flights[1].BestValueScore = 90
	Sort(flights, nil)
	if flights[0].FlightNumber != "A2" {
		t.Fatalf("best value default first = %s, want A2", flights[0].FlightNumber)
	}
}

func numbers(flights []ScoredFlight) []string {
	out := make([]string, len(flights))
	for i, f := range flights {
		out[i] = f.FlightNumber
	}
	return out
}
