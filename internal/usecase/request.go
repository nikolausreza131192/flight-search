package usecase

// SearchRequest describes an inbound flight search query.
type SearchRequest struct {
	Origin        string         `json:"origin"`
	Destination   string         `json:"destination"`
	DepartureDate string         `json:"departureDate"`
	ReturnDate    *string        `json:"returnDate,omitempty"`
	Passengers    int            `json:"passengers,omitempty"`
	CabinClass    string         `json:"cabinClass,omitempty"`
	Filters       *Filters       `json:"filters,omitempty"`
	Sort          *SortSpec      `json:"sort,omitempty"`
	MultiCity     []MultiCityLeg `json:"multiCity,omitempty"`
}

// MultiCityLeg is a single leg of a multi-city search.
type MultiCityLeg struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departureDate"`
}

// Filters narrows down the aggregated result set. Every field is optional.
type Filters struct {
	MinPrice           *float64 `json:"minPrice,omitempty"`
	MaxPrice           *float64 `json:"maxPrice,omitempty"`
	MaxStops           *int     `json:"maxStops,omitempty"`
	Airlines           []string `json:"airlines,omitempty"`
	DepartureTimeFrom  *string  `json:"departureTimeFrom,omitempty"`
	DepartureTimeTo    *string  `json:"departureTimeTo,omitempty"`
	ArrivalTimeFrom    *string  `json:"arrivalTimeFrom,omitempty"`
	ArrivalTimeTo      *string  `json:"arrivalTimeTo,omitempty"`
	MaxDurationMinutes *int     `json:"maxDurationMinutes,omitempty"`
}

// SortSpec controls result ordering.
type SortSpec struct {
	By    string `json:"by"`    // best_value | price | duration | departure_time | arrival_time
	Order string `json:"order"` // asc | desc
}

// Supported sort keys.
const (
	SortBestValue     = "best_value"
	SortPrice         = "price"
	SortDuration      = "duration"
	SortDepartureTime = "departure_time"
	SortArrivalTime   = "arrival_time"
)
