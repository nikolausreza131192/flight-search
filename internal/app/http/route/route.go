package route

import (
	"net/http"

	"bookcabin/internal/app/http/handler"
)

// New builds the application router for the given handler.
func New(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/flights/search", h.SearchFlights)

	return Recoverer(RequestLogger(mux))
}
