// Package response provides the standard JSON envelope helpers used by every HTTP
// handler — see library-docs/07-api/guidelines.md ("Error format") and
// library-docs/05-architecture/cross-cutting.md.
package response

import (
	"encoding/json"
	"net/http"
)

// ErrorBody matches library-docs/07-api/contracts/openapi/_shared.yaml#/components/schemas/ErrorResponse.
type ErrorBody struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId,omitempty"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Error(w http.ResponseWriter, status int, code, message, correlationID string) {
	JSON(w, status, ErrorBody{Error: code, Message: message, CorrelationID: correlationID})
}
