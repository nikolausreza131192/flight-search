// Package entity is the innermost layer of the architecture. It holds the
// enterprise entities and value objects of the flight search domain and has no
// dependency on any other package of this project.
package entity

// Airline identifies the operating carrier.
type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Endpoint is a departure or arrival point of a flight.
type Endpoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

// Duration is a trip duration in minutes plus a human readable form.
type Duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

// Price is a fare amount with its currency and a display representation.
type Price struct {
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Formatted string `json:"formatted"`
}

// Baggage holds normalized baggage allowance information.
type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

// Segment is a single leg of a connecting flight.
type Segment struct {
	FlightNumber    string   `json:"flight_number"`
	Departure       Endpoint `json:"departure"`
	Arrival         Endpoint `json:"arrival"`
	DurationMinutes int      `json:"duration_minutes"`
	LayoverMinutes  int      `json:"layover_minutes,omitempty"`
}

// Flight is the single, provider-agnostic representation that every airline
// adapter normalizes its response into. Ranking/presentation concerns (such as
// a best value score) deliberately live outside of this entity.
type Flight struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	Airline        Airline   `json:"airline"`
	FlightNumber   string    `json:"flight_number"`
	Departure      Endpoint  `json:"departure"`
	Arrival        Endpoint  `json:"arrival"`
	Duration       Duration  `json:"duration"`
	Stops          int       `json:"stops"`
	Price          Price     `json:"price"`
	AvailableSeats int       `json:"available_seats"`
	CabinClass     string    `json:"cabin_class"`
	Aircraft       *string   `json:"aircraft"`
	Amenities      []string  `json:"amenities"`
	Baggage        Baggage   `json:"baggage"`
	Segments       []Segment `json:"segments,omitempty"`
}
