package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// timeFormat is used for every timestamp field in JSON responses, matching
// the `format: date-time` (RFC 3339) fields in openapi.yaml.
const timeFormat = time.RFC3339

func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func RespondError(w http.ResponseWriter, status int, message, code string) {
	RespondJSON(w, status, errorResponse{Error: message, Code: code})
}

// parseUUIDParam parses the named chi URL parameter as a UUID. On failure
// it writes a 400 ERR_VALIDATION response (using label, e.g. "board id",
// in the message) and returns ok=false — callers should return immediately
// when ok is false.
func parseUUIDParam(w http.ResponseWriter, r *http.Request, paramName, label string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, paramName))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid "+label, "ERR_VALIDATION")
		return uuid.UUID{}, false
	}
	return id, true
}

// decodeJSON decodes the request body into dst. On failure it writes a 400
// ERR_VALIDATION response and returns ok=false — callers should return
// immediately when ok is false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body", "ERR_VALIDATION")
		return false
	}
	return true
}
