package providers

import (
	"time"

	"bookcabin/internal/entity"
	"bookcabin/internal/util/currency"
	"bookcabin/internal/util/timeutil"
)

// finalize fills in derived fields that every provider shares: the display
// duration, the display price, the composite id and a non-nil amenities slice.
func finalize(f *entity.Flight) {
	f.ID = f.FlightNumber + "_" + f.Provider
	f.Duration.Formatted = timeutil.FormatDuration(f.Duration.TotalMinutes)
	f.Price.Formatted = currency.Format(f.Price.Amount, f.Price.Currency)
	if f.Amenities == nil {
		f.Amenities = []string{}
	}
}

// endpoint builds a normalized endpoint from a parsed time and airport code.
func endpoint(code, providedCity string, t time.Time) entity.Endpoint {
	return entity.Endpoint{
		Airport:   code,
		City:      entity.CityFor(code, providedCity),
		Datetime:  timeutil.Format(t),
		Timestamp: t.Unix(),
	}
}

// valid reports whether an arrival/departure pair is chronologically sound.
func valid(dep, arr time.Time) bool {
	return arr.After(dep)
}

// adapt aligns a provider's normalized flights with the incoming criteria.
//
// The fixture payloads only describe the CGK -> DPS route on a single
// date. To keep every search mode (one-way, round-trip, multi-city) demonstrable
// this helper:
//
//   - keeps flights that match the requested origin/destination, and mirrors
//     them when the request is the reverse route (so DPS -> CGK works);
//   - re-bases the departure/arrival onto the requested departure date while
//     preserving the original clock time, timezone and trip duration.
func adapt(in []entity.Flight, criteria entity.SearchCriteria) []entity.Flight {
	base, baseErr := time.Parse(timeutil.DateLayout, criteria.DepartureDate)

	out := make([]entity.Flight, 0, len(in))
	for _, f := range in {
		forward := f.Departure.Airport == criteria.Origin && f.Arrival.Airport == criteria.Destination
		reverse := f.Departure.Airport == criteria.Destination && f.Arrival.Airport == criteria.Origin
		if !forward && !reverse {
			continue
		}
		if reverse {
			f = mirror(f)
		}
		if baseErr == nil {
			f = rebase(f, base)
		}
		out = append(out, f)
	}
	return out
}

// rebase moves a flight onto the given date, keeping the clock time, timezone
// and duration intact.
func rebase(f entity.Flight, base time.Time) entity.Flight {
	dep, err1 := time.Parse(time.RFC3339, f.Departure.Datetime)
	arr, err2 := time.Parse(time.RFC3339, f.Arrival.Datetime)
	if err1 != nil || err2 != nil {
		return f
	}
	duration := arr.Sub(dep)
	newDep := timeutil.AtTimeOnDate(base, dep)
	newArr := newDep.Add(duration).In(arr.Location())
	f.Departure.Datetime = timeutil.Format(newDep)
	f.Departure.Timestamp = newDep.Unix()
	f.Arrival.Datetime = timeutil.Format(newArr)
	f.Arrival.Timestamp = newArr.Unix()
	return f
}

// mirror produces the reverse-direction version of a flight. The new departure
// occurs at the original arrival point/time and the arrival is recomputed from
// the preserved duration, so the resulting times remain valid.
func mirror(f entity.Flight) entity.Flight {
	dep, err1 := time.Parse(time.RFC3339, f.Departure.Datetime)
	arr, err2 := time.Parse(time.RFC3339, f.Arrival.Datetime)
	if err1 == nil && err2 == nil {
		duration := arr.Sub(dep)
		newDep := time.Date(arr.Year(), arr.Month(), arr.Day(), arr.Hour(), arr.Minute(), arr.Second(), 0, arr.Location())
		newArr := newDep.Add(duration).In(dep.Location())

		origin := f.Departure
		f.Departure = entity.Endpoint{
			Airport:   f.Arrival.Airport,
			City:      f.Arrival.City,
			Datetime:  timeutil.Format(newDep),
			Timestamp: newDep.Unix(),
		}
		f.Arrival = entity.Endpoint{
			Airport:   origin.Airport,
			City:      origin.City,
			Datetime:  timeutil.Format(newArr),
			Timestamp: newArr.Unix(),
		}
	}
	// Segment detail for a mirrored flight is not meaningful, drop it.
	f.Segments = nil
	return f
}
