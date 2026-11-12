# Flight Schedule Search & Aggregation API

## Approach

- **Clean Architecture.** Dependencies point inward: `app/http` and
  `repository` depend on `usecase`, which depends only on `entity`. Outer layers
  plug into interfaces declared by the use case, so providers and the
  HTTP surface can evolve independently.
- **Interfaces & adapters.** Every airline is an adapter behind the
  `usecase.AirlineProvider` interface, owning its own response contract, normalizer,
  rate limit and failure profile. In this repo the adapters read mock payloads
  from `template/` instead of calling live APIs.
- **Composition root + DI.** `internal/app/dependency` uses Google Wire to build
  the object graph: airline providers → search repository → cache/service →
  handler → router.
- **Reliability.** Providers are queried in parallel with per-provider retries
  (exponential backoff + jitter) and an overall timeout. Full responses are cached
  by request key.
- **Ranking.** Each flight gets a `best_value_score` in `[0, 100]`, a weighted
  sum of inverse-normalized price, duration and stops (0.5 / 0.3 / 0.2).

## Setup

Requires Go 1.25+.

```bash
go build ./...
go run ./cmd/http        # listens on :8080
```

Run tests:

```bash
go test ./...
```

Search (one-way):

```bash
curl -s -X POST http://localhost:8080/api/v1/flights/search \
  -H 'Content-Type: application/json' \
  -d '{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy"}'
```

Search (one-way with filters & sort):

```bash
curl -s -X POST http://localhost:8080/api/v1/flights/search \
  -H 'Content-Type: application/json' \
  -d '{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","filters":{"maxStops":0,"maxPrice":1000000,"airlines":["QZ"]},"sort":{"by":"price","order":"asc"}}'
```

Search (round-trip):

```bash
curl -s -X POST http://localhost:8080/api/v1/flights/search \
  -H 'Content-Type: application/json' \
  -d '{"origin":"CGK","destination":"DPS","departureDate":"2026-12-16","passengers":2,"cabinClass":"first","returnDate":"2026-12-18"}'
```

Search (multi-city):

```bash
curl -s -X POST http://localhost:8080/api/v1/flights/search \
  -H 'Content-Type: application/json' \
  -d '{
  "multiCity": [
    { "origin": "CGK", "destination": "DPS", "departureDate": "2025-12-15" },
    { "origin": "DPS", "destination": "CGK", "departureDate": "2025-12-20" }
  ],
  "passengers": 1,
  "cabinClass": "economy"
}'
```

Configuration lives in [template/config.json](template/config.json) and is loaded
by `config.Load`:

```json
{
  "addr": ":8080",
  "log_level": "info",
  "cache_ttl": "5m",
  "search_timeout": "3s",
  "max_retries": 2,
  "retry_base_delay": "50ms",
  "simulate_latency": true,
  "airlines": {
    "GI": {
      "search": {
        "file": "garuda_indonesia_search_response.json",
        "rate_limit": 50,
        "rate_limit_window": "1m",
        "min_delay": "50ms",
        "max_delay": "100ms",
        "failure_rate": 0
      }
    }
  }
}
```

| Key                | Description                                              |
|--------------------|----------------------------------------------------------|
| `addr`             | Listen address.                                          |
| `log_level`        | `debug` / `info` / `warn` / `error`.                     |
| `cache_ttl`        | Response cache TTL; `0` disables caching.                |
| `search_timeout`   | Overall deadline for one search.                         |
| `max_retries`      | Retries per provider on failure.                         |
| `retry_base_delay` | Backoff base; grows exponentially with jitter.           |
| `simulate_latency` | Simulate provider latency.                               |
| `airlines`         | Per-airline settings, keyed by IATA code (`GI`, `JT`, `ID`, `QZ`). |

Durations use Go syntax (`5m`, `3s`, `50ms`). Each airline's fixture `file` is
resolved relative to `template/`, so entries are just the payload file name.

To run with custom config, edit `template/config.json`.

## API

`POST /api/v1/flights/search` → aggregate flight results. Body is JSON.

| Field           | Type           | Required | Notes                                           |
|-----------------|----------------|----------|-------------------------------------------------|
| `origin`        | string (IATA)  | yes\*    | e.g. `CGK`.                                     |
| `destination`   | string (IATA)  | yes\*    | e.g. `DPS`.                                     |
| `departureDate` | string         | yes\*    | `YYYY-MM-DD`.                                   |
| `returnDate`    | string \| null | no       | Present ⇒ round-trip.                           |
| `passengers`    | int            | no       | Default `1`; fare is multiplied by this.        |
| `cabinClass`    | string         | no       | Default `economy`.                              |
| `filters`       | object         | no       | `minPrice`, `maxPrice`, `maxStops`, `airlines`, |
|                 |                |          | `departureTimeFrom/To`, `arrivalTimeFrom/To`,   |
|                 |                |          | `maxDurationMinutes`.                           |
| `sort`          | object         | no       | `{"by": "...", "order": "asc\|desc"}`; `by` is  |
|                 |                |          | `best_value`, `price`, `duration`,              |
|                 |                |          | `departure_time` or `arrival_time`.             |
| `multiCity`     | array of legs  | no       | Takes precedence over one-way/round-trip.       |

\* For `multiCity`, supply `origin`/`destination`/`departureDate` per leg.

**`filters` and `sort` are ignored for round-trip.** When `returnDate` is set,
both fields are dropped and `itineraries` are always ranked by `best_value`. They
apply to the flights for a one-way search, and to each leg for a multi-city search.

Invalid JSON or validation failures return `400` with `{"error":"..."}`.

## Sample request & response

One-way request:

```json
{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "passengers": 1,
  "cabinClass": "economy"
}
```

Response (one flight shown; see [template/expected_result.json](template/expected_result.json)):

```json
{
  "search_criteria": {
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "passengers": 1,
    "cabin_class": "economy"
  },
  "metadata": {
    "total_results": 15,
    "providers_queried": 4,
    "providers_succeeded": 4,
    "providers_failed": 0,
    "search_time_ms": 285,
    "cache_hit": false
  },
  "flights": [
    {
      "id": "QZ7250_AirAsia",
      "provider": "AirAsia",
      "airline": { "name": "AirAsia", "code": "QZ" },
      "flight_number": "QZ7250",
      "departure": {
        "airport": "CGK",
        "city": "Jakarta",
        "datetime": "2025-12-15T15:15:00+07:00",
        "timestamp": 1734246900
      },
      "arrival": {
        "airport": "DPS",
        "city": "Denpasar",
        "datetime": "2025-12-15T20:35:00+08:00",
        "timestamp": 1734267300
      },
      "duration": { "total_minutes": 260, "formatted": "4h 20m" },
      "stops": 1,
      "price": { "amount": 485000, "currency": "IDR", "formatted": "Rp 485.000" },
      "available_seats": 88,
      "cabin_class": "economy",
      "aircraft": null,
      "amenities": [],
      "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" },
      "best_value_score": 100
    }
  ]
}
```

Round-trip request (`returnDate` turns a one-way search into a round-trip):

```json
{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "returnDate": "2025-12-20",
  "passengers": 2,
  "cabinClass": "economy"
}
```

The response adds an `itineraries` array pairing each departure journey with each return journey. 
`flights` holds the departure journey, and `metadata.total_results` is the number of itineraries (see
[why](#why-total_results-counts-itineraries)). `filters` and `sort` do not
apply to round-trip. Itineraries are always ranked by `best_value`:

```json
{
  "search_criteria": {
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 2,
    "cabin_class": "economy"
  },
  "metadata": {
    "total_results": 100,
    "providers_queried": 4,
    "providers_succeeded": 4,
    "providers_failed": 0,
    "search_time_ms": 2,
    "cache_hit": false
  },
  "flights": [
    {
      "id": "QZ532_AirAsia",
      "provider": "AirAsia",
      "airline": { "name": "AirAsia", "code": "QZ" },
      "flight_number": "QZ532",
      "departure": {
        "airport": "CGK",
        "city": "Jakarta",
        "datetime": "2025-12-15T19:30:00+07:00",
        "timestamp": 1765801800
      },
      "arrival": {
        "airport": "DPS",
        "city": "Denpasar",
        "datetime": "2025-12-15T22:10:00+08:00",
        "timestamp": 1765807800
      },
      "duration": { "total_minutes": 100, "formatted": "1h 40m" },
      "stops": 0,
      "price": { "amount": 1190000, "currency": "IDR", "formatted": "Rp 1.190.000" },
      "available_seats": 72,
      "cabin_class": "economy",
      "aircraft": null,
      "amenities": [],
      "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" },
      "best_value_score": 95.97
    }
  ],
  "itineraries": [
    {
      "id": "QZ532_AirAsia::QZ532_AirAsia",
      "trip_type": "round_trip",
      "departure_journey": {
        "id": "QZ532_AirAsia",
        "provider": "AirAsia",
        "airline": { "name": "AirAsia", "code": "QZ" },
        "flight_number": "QZ532",
        "departure": {
          "airport": "CGK",
          "city": "Jakarta",
          "datetime": "2025-12-15T19:30:00+07:00",
          "timestamp": 1765801800
        },
        "arrival": {
          "airport": "DPS",
          "city": "Denpasar",
          "datetime": "2025-12-15T22:10:00+08:00",
          "timestamp": 1765807800
        },
        "duration": { "total_minutes": 100, "formatted": "1h 40m" },
        "stops": 0,
        "price": { "amount": 1190000, "currency": "IDR", "formatted": "Rp 1.190.000" },
        "available_seats": 72,
        "cabin_class": "economy",
        "aircraft": null,
        "amenities": [],
        "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" }
      },
      "return_journey": {
        "id": "QZ532_AirAsia",
        "provider": "AirAsia",
        "airline": { "name": "AirAsia", "code": "QZ" },
        "flight_number": "QZ532",
        "departure": {
          "airport": "DPS",
          "city": "Denpasar",
          "datetime": "2025-12-20T22:10:00+08:00",
          "timestamp": 1766239800
        },
        "arrival": {
          "airport": "CGK",
          "city": "Jakarta",
          "datetime": "2025-12-20T22:50:00+07:00",
          "timestamp": 1766245800
        },
        "duration": { "total_minutes": 100, "formatted": "1h 40m" },
        "stops": 0,
        "price": { "amount": 1190000, "currency": "IDR", "formatted": "Rp 1.190.000" },
        "available_seats": 72,
        "cabin_class": "economy",
        "aircraft": null,
        "amenities": [],
        "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" }
      },
      "total_price": { "amount": 2380000, "currency": "IDR", "formatted": "Rp 2.380.000" },
      "total_duration": { "total_minutes": 200, "formatted": "3h 20m" },
      "stops": 0,
      "best_value_score": 97.18
    },
    {
      "id": "QZ532_AirAsia::QZ520_AirAsia",
      "trip_type": "round_trip",
      "departure_journey": {
        "id": "QZ532_AirAsia",
        "provider": "AirAsia",
        "airline": { "name": "AirAsia", "code": "QZ" },
        "flight_number": "QZ532",
        "departure": {
          "airport": "CGK",
          "city": "Jakarta",
          "datetime": "2025-12-15T19:30:00+07:00",
          "timestamp": 1765801800
        },
        "arrival": {
          "airport": "DPS",
          "city": "Denpasar",
          "datetime": "2025-12-15T22:10:00+08:00",
          "timestamp": 1765807800
        },
        "duration": { "total_minutes": 100, "formatted": "1h 40m" },
        "stops": 0,
        "price": { "amount": 1190000, "currency": "IDR", "formatted": "Rp 1.190.000" },
        "available_seats": 72,
        "cabin_class": "economy",
        "aircraft": null,
        "amenities": [],
        "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" }
      },
      "return_journey": {
        "id": "QZ520_AirAsia",
        "provider": "AirAsia",
        "airline": { "name": "AirAsia", "code": "QZ" },
        "flight_number": "QZ520",
        "departure": {
          "airport": "DPS",
          "city": "Denpasar",
          "datetime": "2025-12-20T07:25:00+08:00",
          "timestamp": 1766186700
        },
        "arrival": {
          "airport": "CGK",
          "city": "Jakarta",
          "datetime": "2025-12-20T08:05:00+07:00",
          "timestamp": 1766192700
        },
        "duration": { "total_minutes": 100, "formatted": "1h 40m" },
        "stops": 0,
        "price": { "amount": 1300000, "currency": "IDR", "formatted": "Rp 1.300.000" },
        "available_seats": 67,
        "cabin_class": "economy",
        "aircraft": null,
        "amenities": [],
        "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" }
      },
      "total_price": { "amount": 2490000, "currency": "IDR", "formatted": "Rp 2.490.000" },
      "total_duration": { "total_minutes": 200, "formatted": "3h 20m" },
      "stops": 0,
      "best_value_score": 95.77
    }
  ]
}
```

#### Why `total_results` counts itineraries

A round-trip search answers "which *paired* trips can I buy", so the headline
count is the number of pairings, not the number of single flights. Concretely,
for the request above:

| Quantity                          | Value | Source                                            |
|-----------------------------------|-------|---------------------------------------------------|
| Departure flights (`CGK→DPS`)     | 13    | one leg's `flights`                               |
| Return flights (`DPS→CGK`)        | 13    | the other leg's `flights`                         |
| Possible pairs                    | 169   | `13 × 13`                                         |
| Itineraries returned              | 100   | capped at `maxItineraryCombinations`              |
| `metadata.total_results`          | 100   | `len(itineraries)`                                |
| Top-level `flights` length        | 13    | the departure journey only, uncapped              |

So `flights` (13) and `total_results` (100) legitimately differ: the former is
the departure pool, the latter is how many round-trip options are on offer. A
one-way search, by contrast, sets `total_results = len(flights)`, and a
multi-city search sums each leg's `total_results`.


Multi-city request (`multiCity` have higher priority over one-way/round-trip):

```json
{
  "multiCity": [
    { "origin": "CGK", "destination": "DPS", "departureDate": "2025-12-15" },
    { "origin": "DPS", "destination": "CGK", "departureDate": "2025-12-20" }
  ],
  "passengers": 1,
  "cabinClass": "economy"
}
```

The response sets the top-level `flights` to `[]` and returns one entry per leg
in `legs`, each carrying its own `search_criteria`, `metadata` and `flights`:

```json
{
  "search_criteria": {
    "passengers": 1,
    "cabin_class": "economy",
    "multi_city": [
      { "origin": "CGK", "destination": "DPS", "departure_date": "2025-12-15" },
      { "origin": "DPS", "destination": "CGK", "departure_date": "2025-12-20" }
    ]
  },
  "metadata": {
    "total_results": 26,
    "providers_queried": 4,
    "providers_succeeded": 4,
    "providers_failed": 0,
    "search_time_ms": 4,
    "cache_hit": false
  },
  "flights": [],
  "legs": [
    {
      "search_criteria": {
        "origin": "CGK",
        "destination": "DPS",
        "departure_date": "2025-12-15",
        "passengers": 1,
        "cabin_class": "economy"
      },
      "metadata": {
        "total_results": 13,
        "providers_queried": 4,
        "providers_succeeded": 4,
        "providers_failed": 0,
        "search_time_ms": 3,
        "cache_hit": false
      },
      "flights": [
        {
          "id": "QZ532_AirAsia",
          "provider": "AirAsia",
          "airline": { "name": "AirAsia", "code": "QZ" },
          "flight_number": "QZ532",
          "departure": {
            "airport": "CGK",
            "city": "Jakarta",
            "datetime": "2025-12-15T19:30:00+07:00",
            "timestamp": 1765801800
          },
          "arrival": {
            "airport": "DPS",
            "city": "Denpasar",
            "datetime": "2025-12-15T22:10:00+08:00",
            "timestamp": 1765807800
          },
          "duration": { "total_minutes": 100, "formatted": "1h 40m" },
          "stops": 0,
          "price": { "amount": 595000, "currency": "IDR", "formatted": "Rp 595.000" },
          "available_seats": 72,
          "cabin_class": "economy",
          "aircraft": null,
          "amenities": [],
          "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" },
          "best_value_score": 95.97
        }
      ]
    },
    {
      "search_criteria": {
        "origin": "DPS",
        "destination": "CGK",
        "departure_date": "2025-12-20",
        "passengers": 1,
        "cabin_class": "economy"
      },
      "metadata": {
        "total_results": 13,
        "providers_queried": 4,
        "providers_succeeded": 4,
        "providers_failed": 0,
        "search_time_ms": 0,
        "cache_hit": false
      },
      "flights": [
        {
          "id": "QZ532_AirAsia",
          "provider": "AirAsia",
          "airline": { "name": "AirAsia", "code": "QZ" },
          "flight_number": "QZ532",
          "departure": {
            "airport": "DPS",
            "city": "Denpasar",
            "datetime": "2025-12-20T22:10:00+08:00",
            "timestamp": 1766239800
          },
          "arrival": {
            "airport": "CGK",
            "city": "Jakarta",
            "datetime": "2025-12-20T22:50:00+07:00",
            "timestamp": 1766245800
          },
          "duration": { "total_minutes": 100, "formatted": "1h 40m" },
          "stops": 0,
          "price": { "amount": 595000, "currency": "IDR", "formatted": "Rp 595.000" },
          "available_seats": 72,
          "cabin_class": "economy",
          "aircraft": null,
          "amenities": [],
          "baggage": { "carry_on": "Cabin baggage only", "checked": "Additional fee" },
          "best_value_score": 95.97
        }
      ]
    }
  ]
}
```

Both legs are abbreviated to a single flight above.

## Best-value scoring

Every flight carries a `best_value_score` in `[0, 100]` (higher is better), and
`best_value` is the default sort key. It is a weighted blend of price, duration
and stops, defined in [ranking.go](internal/usecase/ranking.go):

```text
best_value_score = round(100 × (0.5 × price + 0.3 × duration + 0.2 × stops), 2)
```

| Factor   | Weight | Normalized as                                        |
|----------|--------|------------------------------------------------------|
| Price    | 0.5    | `(max − price) / (max − min)` over the result set    |
| Duration | 0.3    | `(max − minutes) / (max − min)` over the result set  |
| Stops    | 0.2    | `1 − stops / maxStops` over the result set           |

Key properties:

- **Cheaper/faster/fewer-stops is better.** Each factor is *inverse*-normalized,
  so the cheapest flight scores `1.0` on price and the most expensive `0.0`. The
  same holds for duration, and a nonstop flight scores `1.0` for stops.
- **Bounds come from the current result set, so scores are relative, not
  absolute.** The best flight in a set always approaches `100` and the worst
  approaches `0`, regardless of whether the set is cheap or expensive. Adding or
  removing flights (or changing `filters`) recomputes every score, so the same
  flight can score differently across two searches.
- **Computed after filtering.** The min/max bounds are taken from the filtered,
  passenger-scaled set, so filters shape the ranking baseline too.
- **Itineraries are scored the same way**, via a proxy flight built from
  `total_price`, `total_duration` and summed `stops`
  ([scoreItineraries](internal/usecase/search.go)), so round-trip options rank on
  the identical formula.
- **Weights are constants** (`WeightPrice`, `WeightDuration`,
  `WeightStops`). Tuning the ranking means editing the constants, not config.

#### Example

A one-way `CGK→DPS` search returns 13 flights. The scoring bounds are derived
from that set:

```text
price:    485,000 … 1,850,000 IDR  → spread 1,365,000
duration: 100 … 260 minutes        → spread 160
maxStops: 1
```

Three of those flights, scored step by step:

```text
QZ532_AirAsia    price 595,000    duration 100    stops 0   → 95.97
  price    = (1,850,000 − 595,000) / 1,365,000 = 0.9194
  duration = (260 − 100) / 160                 = 1.0000
  stops    = 1 − 0 / 1                         = 1.0000
  score    = 100 × (0.5×0.9194 + 0.3×1.0000 + 0.2×1.0000) = 95.97

JT742_Lion Air   price 890,000    duration 110    stops 0   → 83.29
  price    = (1,850,000 − 890,000) / 1,365,000 = 0.7033
  duration = (260 − 110) / 160                 = 0.9375
  stops    = 1 − 0 / 1                         = 1.0000
  score    = 100 × (0.5×0.7033 + 0.3×0.9375 + 0.2×1.0000) = 83.29

GA315_Garuda     price 1,850,000  duration 225    stops 1   → 6.56
  price    = (1,850,000 − 1,850,000) / 1,365,000 = 0.0000
  duration = (260 − 225) / 160                   = 0.2188
  stops    = 1 − 1 / 1                           = 0.0000
  score    = 100 × (0.5×0.0000 + 0.3×0.2188 + 0.2×0.0000) = 6.56
```

Note the cheapest flight is **not** the winner. `QZ7250_AirAsia` costs 485,000
but takes 260 minutes with 1 stop, so it scores just `50.00`. Price alone
(weight 0.5) can't outvote zeros on duration and stops. `QZ532_AirAsia` costs
110,000 more yet is nonstop and tied for fastest, so it wins at `95.97`.

## Rate limiting

Rate limit configuration for each airline is in `template/config.json` (`rate_limit`,
`rate_limit_window`). `rate_limit <= 0` disables this feature:

| Airline | Code | Limit   |
|---------|------|---------|
| Garuda  | `GI` | 50 / 1m |
| Lion    | `JT` | 40 / 1m |
| Batik   | `ID` | 20 / 1m |
| AirAsia | `QZ` | 10 / 1m |

**Why a sliding window rather than a token bucket:**

**Exact contract.** A token bucket admits a burst up to its capacity and then
refills at a fixed rate, so within one window it can exceed the nominal limit
(capacity + whatever refilled). When an upstream states its quota as "N per
minute", a sliding window guarantees *no more than N in any trailing W*.
