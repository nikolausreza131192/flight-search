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

// Garuda adapts the Garuda Indonesia search API to the AirlineProvider interface
type Garuda struct {
	baseProvider
}

// NewGaruda builds the Garuda Indonesia adapter from config.
func NewGaruda(cfg config.Config) *Garuda {
	return &Garuda{baseProvider: newBaseProvider("GI", cfg)}
}

// Search reads Garuda's response, normalizes it and aligns it with the criteria.
func (g *Garuda) Search(ctx context.Context, criteria entity.SearchCriteria) ([]entity.Flight, error) {
	data, err := g.read(ctx)
	if err != nil {
		return nil, fmt.Errorf("garuda: %w", err)
	}

	flights, err := g.normalize(data)
	if err != nil {
		return nil, err
	}
	return adapt(flights, criteria), nil
}

// Garuda Indonesia response shape
type garudaResponse struct {
	Status  string         `json:"status"`
	Flights []garudaFlight `json:"flights"`
}

type garudaFlight struct {
	FlightID        string          `json:"flight_id"`
	Airline         string          `json:"airline"`
	AirlineCode     string          `json:"airline_code"`
	Departure       garudaEndpoint  `json:"departure"`
	Arrival         garudaEndpoint  `json:"arrival"`
	DurationMinutes int             `json:"duration_minutes"`
	Stops           int             `json:"stops"`
	Aircraft        string          `json:"aircraft"`
	Price           garudaPrice     `json:"price"`
	AvailableSeats  int             `json:"available_seats"`
	FareClass       string          `json:"fare_class"`
	Baggage         garudaBaggage   `json:"baggage"`
	Amenities       []string        `json:"amenities"`
	Segments        []garudaSegment `json:"segments"`
}

type garudaEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type garudaPrice struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type garudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type garudaSegment struct {
	FlightNumber    string         `json:"flight_number"`
	Departure       garudaSegPoint `json:"departure"`
	Arrival         garudaSegPoint `json:"arrival"`
	DurationMinutes int            `json:"duration_minutes"`
	LayoverMinutes  int            `json:"layover_minutes"`
}

type garudaSegPoint struct {
	Airport string `json:"airport"`
	Time    string `json:"time"`
}

func (g *Garuda) normalize(data []byte) ([]entity.Flight, error) {
	var resp garudaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("garuda: decode response: %w", err)
	}

	out := make([]entity.Flight, 0, len(resp.Flights))
	for _, raw := range resp.Flights {
		dep, err := timeutil.Parse(raw.Departure.Time)
		if err != nil {
			return nil, fmt.Errorf("garuda %s: departure: %w", raw.FlightID, err)
		}
		arr, err := timeutil.Parse(raw.Arrival.Time)
		if err != nil {
			return nil, fmt.Errorf("garuda %s: arrival: %w", raw.FlightID, err)
		}

		stops := raw.Stops
		arrCode := raw.Arrival.Airport
		var segments []entity.Segment

		// A connecting flight exposes the true destination through its segments.
		if len(raw.Segments) > 0 {
			last := raw.Segments[len(raw.Segments)-1]
			lastArr, segErr := timeutil.Parse(last.Arrival.Time)
			if segErr == nil {
				arr = lastArr
				arrCode = last.Arrival.Airport
			}
			stops = len(raw.Segments) - 1
			segments = make([]entity.Segment, 0, len(raw.Segments))
			for _, s := range raw.Segments {
				seg, segErr := g.segment(s)
				if segErr != nil {
					return nil, fmt.Errorf("garuda %s: %w", raw.FlightID, segErr)
				}
				segments = append(segments, seg)
			}
		}

		if !valid(dep, arr) {
			// Skip flights whose arrival precedes their departure rather than
			// failing the entire provider response.
			continue
		}

		flight := entity.Flight{
			Provider:     "Garuda Indonesia",
			Airline:      entity.Airline{Name: raw.Airline, Code: raw.AirlineCode},
			FlightNumber: raw.FlightID,
			Departure:    endpoint(raw.Departure.Airport, raw.Departure.City, dep),
			Arrival:      endpoint(arrCode, "", arr),
			Duration: entity.Duration{
				TotalMinutes: int(arr.Sub(dep).Minutes()),
			},
			Stops:          stops,
			Price:          entity.Price{Amount: raw.Price.Amount, Currency: defaultCurrency(raw.Price.Currency)},
			AvailableSeats: raw.AvailableSeats,
			CabinClass:     strings.ToLower(raw.FareClass),
			Aircraft:       optionalString(raw.Aircraft),
			Amenities:      lowerAll(raw.Amenities),
			Baggage: entity.Baggage{
				CarryOn: formatBagCount(raw.Baggage.CarryOn),
				Checked: formatBagCount(raw.Baggage.Checked),
			},
			Segments: segments,
		}
		finalize(&flight)
		out = append(out, flight)
	}
	return out, nil
}

func (g *Garuda) segment(s garudaSegment) (entity.Segment, error) {
	dep, err := timeutil.Parse(s.Departure.Time)
	if err != nil {
		return entity.Segment{}, fmt.Errorf("segment %s departure: %w", s.FlightNumber, err)
	}
	arr, err := timeutil.Parse(s.Arrival.Time)
	if err != nil {
		return entity.Segment{}, fmt.Errorf("segment %s arrival: %w", s.FlightNumber, err)
	}
	return entity.Segment{
		FlightNumber:    s.FlightNumber,
		Departure:       endpoint(s.Departure.Airport, "", dep),
		Arrival:         endpoint(s.Arrival.Airport, "", arr),
		DurationMinutes: s.DurationMinutes,
		LayoverMinutes:  s.LayoverMinutes,
	}, nil
}
