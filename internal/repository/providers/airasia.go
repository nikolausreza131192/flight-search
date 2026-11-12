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

// AirAsia adapts the AirAsia search API to the AirlineProvider interface
type AirAsia struct {
	baseProvider
}

// NewAirAsia builds the AirAsia adapter from config.
func NewAirAsia(cfg config.Config) *AirAsia {
	return &AirAsia{baseProvider: newBaseProvider("QZ", cfg)}
}

// Search reads AirAsia's response, normalizes it and aligns it with the criteria.
func (a *AirAsia) Search(ctx context.Context, criteria entity.SearchCriteria) ([]entity.Flight, error) {
	data, err := a.read(ctx)
	if err != nil {
		return nil, fmt.Errorf("airasia: %w", err)
	}

	flights, err := a.normalize(data)
	if err != nil {
		return nil, err
	}
	return adapt(flights, criteria), nil
}

// AirAsia response shape
type airasiaResponse struct {
	Status  string          `json:"status"`
	Flights []airasiaFlight `json:"flights"`
}

type airasiaFlight struct {
	FlightCode    string  `json:"flight_code"`
	Airline       string  `json:"airline"`
	FromAirport   string  `json:"from_airport"`
	ToAirport     string  `json:"to_airport"`
	DepartTime    string  `json:"depart_time"`
	ArriveTime    string  `json:"arrive_time"`
	DurationHours float64 `json:"duration_hours"`
	DirectFlight  bool    `json:"direct_flight"`
	Stops         []struct {
		Airport         string `json:"airport"`
		WaitTimeMinutes int    `json:"wait_time_minutes"`
	} `json:"stops"`
	PriceIDR    int64  `json:"price_idr"`
	Seats       int    `json:"seats"`
	CabinClass  string `json:"cabin_class"`
	BaggageNote string `json:"baggage_note"`
}

func (a *AirAsia) normalize(data []byte) ([]entity.Flight, error) {
	var resp airasiaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("airasia: decode response: %w", err)
	}

	out := make([]entity.Flight, 0, len(resp.Flights))
	for _, raw := range resp.Flights {
		dep, err := timeutil.Parse(raw.DepartTime)
		if err != nil {
			return nil, fmt.Errorf("airasia %s: departure: %w", raw.FlightCode, err)
		}
		arr, err := timeutil.Parse(raw.ArriveTime)
		if err != nil {
			return nil, fmt.Errorf("airasia %s: arrival: %w", raw.FlightCode, err)
		}
		if !valid(dep, arr) {
			continue
		}

		carryOn, checked := a.baggage(raw.BaggageNote)

		flight := entity.Flight{
			Provider:     "AirAsia",
			Airline:      entity.Airline{Name: raw.Airline, Code: a.code},
			FlightNumber: raw.FlightCode,
			Departure:    endpoint(raw.FromAirport, "", dep),
			Arrival:      endpoint(raw.ToAirport, "", arr),
			Duration: entity.Duration{
				TotalMinutes: int(arr.Sub(dep).Minutes()),
			},
			Stops:          len(raw.Stops),
			Price:          entity.Price{Amount: raw.PriceIDR, Currency: "IDR"},
			AvailableSeats: raw.Seats,
			CabinClass:     strings.ToLower(raw.CabinClass),
			Aircraft:       nil,
			Amenities:      []string{},
			Baggage:        entity.Baggage{CarryOn: carryOn, Checked: checked},
		}
		finalize(&flight)
		out = append(out, flight)
	}
	return out, nil
}

// baggage translates AirAsia's free-text baggage note into structured output.
func (a *AirAsia) baggage(note string) (string, string) {
	parts := strings.Split(note, ",")
	carryOn := strings.TrimSpace(parts[0])
	checked := ""
	if len(parts) > 1 {
		second := strings.TrimSpace(parts[1])
		lower := strings.ToLower(second)
		switch {
		case strings.Contains(lower, "additional fee"), strings.Contains(lower, "additional"):
			checked = "Additional fee"
		case strings.Contains(lower, "checked"):
			checked = cleanBagPart(second, "checked")
		default:
			checked = second
		}
	}
	return carryOn, checked
}
