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

// Batik adapts the Batik Air search API to the AirlineProvider interface
type Batik struct {
	baseProvider
}

// NewBatik builds the Batik Air adapter from config.
func NewBatik(cfg config.Config) *Batik {
	return &Batik{baseProvider: newBaseProvider("ID", cfg)}
}

// Search reads Batik's response, normalizes it and aligns it with the criteria.
func (b *Batik) Search(ctx context.Context, criteria entity.SearchCriteria) ([]entity.Flight, error) {
	data, err := b.read(ctx)
	if err != nil {
		return nil, fmt.Errorf("batik air: %w", err)
	}

	flights, err := b.normalize(data)
	if err != nil {
		return nil, err
	}
	return adapt(flights, criteria), nil
}

// Batik Air response shape
type batikResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Results []batikFlight `json:"results"`
}

type batikFlight struct {
	FlightNumber      string `json:"flightNumber"`
	AirlineName       string `json:"airlineName"`
	AirlineIATA       string `json:"airlineIATA"`
	Origin            string `json:"origin"`
	Destination       string `json:"destination"`
	DepartureDateTime string `json:"departureDateTime"`
	ArrivalDateTime   string `json:"arrivalDateTime"`
	TravelTime        string `json:"travelTime"`
	NumberOfStops     int    `json:"numberOfStops"`
	Fare              struct {
		BasePrice    int64  `json:"basePrice"`
		Taxes        int64  `json:"taxes"`
		TotalPrice   int64  `json:"totalPrice"`
		CurrencyCode string `json:"currencyCode"`
		Class        string `json:"class"`
	} `json:"fare"`
	SeatsAvailable  int      `json:"seatsAvailable"`
	AircraftModel   string   `json:"aircraftModel"`
	BaggageInfo     string   `json:"baggageInfo"`
	OnboardServices []string `json:"onboardServices"`
}

// fareClassNames maps IATA cabin letters to a friendly cabin name.
var fareClassNames = map[string]string{
	"F": "first",
	"J": "business",
	"C": "business",
	"W": "premium_economy",
	"Y": "economy",
	"M": "economy",
}

func (b *Batik) normalize(data []byte) ([]entity.Flight, error) {
	var resp batikResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("batik air: decode response: %w", err)
	}

	out := make([]entity.Flight, 0, len(resp.Results))
	for _, raw := range resp.Results {
		dep, err := timeutil.Parse(raw.DepartureDateTime)
		if err != nil {
			return nil, fmt.Errorf("batik air %s: departure: %w", raw.FlightNumber, err)
		}
		arr, err := timeutil.Parse(raw.ArrivalDateTime)
		if err != nil {
			return nil, fmt.Errorf("batik air %s: arrival: %w", raw.FlightNumber, err)
		}
		if !valid(dep, arr) {
			continue
		}

		carryOn, checked := b.parseBaggageNote(raw.BaggageInfo)

		flight := entity.Flight{
			Provider:     "Batik Air",
			Airline:      entity.Airline{Name: raw.AirlineName, Code: raw.AirlineIATA},
			FlightNumber: raw.FlightNumber,
			Departure:    endpoint(raw.Origin, "", dep),
			Arrival:      endpoint(raw.Destination, "", arr),
			Duration: entity.Duration{
				TotalMinutes: int(arr.Sub(dep).Minutes()),
			},
			Stops:          raw.NumberOfStops,
			Price:          entity.Price{Amount: raw.Fare.TotalPrice, Currency: defaultCurrency(raw.Fare.CurrencyCode)},
			AvailableSeats: raw.SeatsAvailable,
			CabinClass:     b.cabinClass(raw.Fare.Class),
			Aircraft:       optionalString(raw.AircraftModel),
			Amenities:      lowerAll(raw.OnboardServices),
			Baggage:        entity.Baggage{CarryOn: carryOn, Checked: checked},
		}
		finalize(&flight)
		out = append(out, flight)
	}
	return out, nil
}

func (b *Batik) cabinClass(code string) string {
	upper := strings.ToUpper(strings.TrimSpace(code))
	if name, ok := fareClassNames[upper]; ok {
		return name
	}
	return strings.ToLower(strings.TrimSpace(code))
}

// parseBaggageNote splits Batik's free-text note such as
// "7kg cabin, 20kg checked" into carry-on and checked components.
func (b *Batik) parseBaggageNote(note string) (string, string) {
	var carryOn, checked string
	for _, part := range strings.Split(note, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		lower := strings.ToLower(p)
		switch {
		case strings.Contains(lower, "cabin"), strings.Contains(lower, "carry"):
			carryOn = cleanBagPart(p, "cabin", "carry-on", "carry on")
		case strings.Contains(lower, "checked"), strings.Contains(lower, "hold"):
			checked = cleanBagPart(p, "checked", "hold")
		default:
			if carryOn == "" {
				carryOn = p
			} else if checked == "" {
				checked = p
			}
		}
	}
	return carryOn, checked
}
