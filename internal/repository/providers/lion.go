package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bookcabin/config"
	"bookcabin/internal/entity"
	"bookcabin/internal/util/timeutil"
)

// Lion adapts the Lion Air search API to the AirlineProvider interface
type Lion struct {
	baseProvider
}

// NewLion builds the Lion Air adapter from config.
func NewLion(cfg config.Config) *Lion {
	return &Lion{baseProvider: newBaseProvider("JT", cfg)}
}

// Search reads Lion's response, normalizes it and aligns it with the criteria.
func (l *Lion) Search(ctx context.Context, criteria entity.SearchCriteria) ([]entity.Flight, error) {
	data, err := l.read(ctx)
	if err != nil {
		return nil, fmt.Errorf("lion air: %w", err)
	}

	flights, err := l.normalize(data)
	if err != nil {
		return nil, err
	}
	return adapt(flights, criteria), nil
}

// Lion Air response shape
type lionResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AvailableFlights []lionFlight `json:"available_flights"`
	} `json:"data"`
}

type lionFlight struct {
	ID      string `json:"id"`
	Carrier struct {
		Name string `json:"name"`
		IATA string `json:"iata"`
	} `json:"carrier"`
	Route struct {
		From lionPlace `json:"from"`
		To   lionPlace `json:"to"`
	} `json:"route"`
	Schedule struct {
		Departure         string `json:"departure"`
		DepartureTimezone string `json:"departure_timezone"`
		Arrival           string `json:"arrival"`
		ArrivalTimezone   string `json:"arrival_timezone"`
	} `json:"schedule"`
	FlightTime int  `json:"flight_time"`
	IsDirect   bool `json:"is_direct"`
	StopCount  int  `json:"stop_count"`
	Pricing    struct {
		Total    int64  `json:"total"`
		Currency string `json:"currency"`
		FareType string `json:"fare_type"`
	} `json:"pricing"`
	SeatsLeft int    `json:"seats_left"`
	PlaneType string `json:"plane_type"`
	Services  struct {
		WifiAvailable    bool `json:"wifi_available"`
		MealsIncluded    bool `json:"meals_included"`
		BaggageAllowance struct {
			Cabin string `json:"cabin"`
			Hold  string `json:"hold"`
		} `json:"baggage_allowance"`
	} `json:"services"`
}

type lionPlace struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

func (l *Lion) normalize(data []byte) ([]entity.Flight, error) {
	var resp lionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("lion air: decode response: %w", err)
	}

	out := make([]entity.Flight, 0, len(resp.Data.AvailableFlights))
	for _, raw := range resp.Data.AvailableFlights {
		dep, err := timeutil.ParseInLocation(raw.Schedule.Departure, raw.Schedule.DepartureTimezone)
		if err != nil {
			return nil, fmt.Errorf("lion air %s: departure: %w", raw.ID, err)
		}
		arr, err := timeutil.ParseInLocation(raw.Schedule.Arrival, raw.Schedule.ArrivalTimezone)
		if err != nil {
			return nil, fmt.Errorf("lion air %s: arrival: %w", raw.ID, err)
		}
		if !valid(dep, arr) {
			continue
		}

		stops := 0
		if !raw.IsDirect {
			stops = raw.StopCount
		}

		amenities := []string{}
		if raw.Services.WifiAvailable {
			amenities = append(amenities, "wifi")
		}
		if raw.Services.MealsIncluded {
			amenities = append(amenities, "meal")
		}

		flight := entity.Flight{
			Provider:     "Lion Air",
			Airline:      entity.Airline{Name: raw.Carrier.Name, Code: raw.Carrier.IATA},
			FlightNumber: raw.ID,
			Departure:    endpoint(raw.Route.From.Code, raw.Route.From.City, dep),
			Arrival:      endpoint(raw.Route.To.Code, raw.Route.To.City, arr),
			Duration: entity.Duration{
				TotalMinutes: int(arr.Sub(dep).Minutes()),
			},
			Stops:          stops,
			Price:          entity.Price{Amount: raw.Pricing.Total, Currency: defaultCurrency(raw.Pricing.Currency)},
			AvailableSeats: raw.SeatsLeft,
			CabinClass:     strings.ToLower(raw.Pricing.FareType),
			Aircraft:       optionalString(raw.PlaneType),
			Amenities:      amenities,
			Baggage: entity.Baggage{
				CarryOn: raw.Services.BaggageAllowance.Cabin,
				Checked: raw.Services.BaggageAllowance.Hold,
			},
		}
		finalize(&flight)
		out = append(out, flight)
	}
	return out, nil
}
