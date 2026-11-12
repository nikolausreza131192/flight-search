package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"bookcabin/internal/entity"
	"bookcabin/internal/usecase"
)

type Handler struct {
	search usecase.SearchItf
}

func New(search usecase.SearchItf) *Handler {
	return &Handler{search: search}
}

// SearchFlights handles flight search requests.
func (h *Handler) SearchFlights(w http.ResponseWriter, r *http.Request) {
	var req usecase.SearchRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request body: "+err.Error())
		return
	}

	resp, err := h.search.Search(r.Context(), req)
	if err != nil {
		if errors.Is(err, entity.ErrValidation) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("flight search failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ErrorResponse is the JSON body returned for all error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteError writes a JSON error body. It is exported so middleware in the
// route package can produce responses with the same shape.
func WriteError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func writeError(w http.ResponseWriter, status int, message string) {
	WriteError(w, status, message)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
