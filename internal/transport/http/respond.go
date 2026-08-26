package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

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
