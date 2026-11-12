package entity

// SearchCriteria is the normalized origin/destination/date value object
type SearchCriteria struct {
	Origin        string         `json:"origin,omitempty"`
	Destination   string         `json:"destination,omitempty"`
	DepartureDate string         `json:"departure_date,omitempty"`
	ReturnDate    string         `json:"return_date,omitempty"`
	Passengers    int            `json:"passengers"`
	CabinClass    string         `json:"cabin_class"`
	MultiCity     []MultiCityLeg `json:"multi_city,omitempty"`
}

// MultiCityLeg is one leg of a multi-city search
type MultiCityLeg struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
}
