// Package httpx contains small shared helpers for writing consistent
// JSON responses across all handlers.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// WriteJSON writes v as a JSON body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: failed to encode json response: %v", err)
	}
}

// ErrorResponse is the standard error envelope used across the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteError writes a standard {"error": "..."} JSON body.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// DecodeJSON decodes the request body into dst. It rejects unknown fields
// so clients get a clear error instead of silently ignored typos.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
