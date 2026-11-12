package entity

// SearchCriteria is the normalized origin/destination/date value object
type SearchCriteria struct {
	Origin        string         `json:"origin,omitempty"`
	Destination   string         `json:"destination,omitempty"`
	DepartureDate string         `json:"departure_date,omitempty"`
	ReturnDate    string         `json:"return_date,omitempty"`
	Passengers    int            `json:"passengers"`
	CabinClass    string         `json:"cabin_class"`
	Filters       *Filters       `json:"filters,omitempty"`
	Sort          *SortSpec      `json:"sort,omitempty"`
	MultiCity     []MultiCityLeg `json:"multi_city,omitempty"`
}

// MultiCityLeg is one leg of a multi-city search
type MultiCityLeg struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
}

// Filters is the filter criteria echoed back on a search. Every field is optional.
type Filters struct {
	MinPrice           *float64 `json:"min_price,omitempty"`
	MaxPrice           *float64 `json:"max_price,omitempty"`
	MaxStops           *int     `json:"max_stops,omitempty"`
	Airlines           []string `json:"airlines,omitempty"`
	DepartureTimeFrom  *string  `json:"departure_time_from,omitempty"`
	DepartureTimeTo    *string  `json:"departure_time_to,omitempty"`
	ArrivalTimeFrom    *string  `json:"arrival_time_from,omitempty"`
	ArrivalTimeTo      *string  `json:"arrival_time_to,omitempty"`
	MaxDurationMinutes *int     `json:"max_duration_minutes,omitempty"`
}

// SortSpec is the sort criteria echoed back on a search.
type SortSpec struct {
	By    string `json:"by"`    // best_value | price | duration | departure_time | arrival_time
	Order string `json:"order"` // asc | desc
}
