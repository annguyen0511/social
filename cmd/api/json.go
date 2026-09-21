package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1 << 20 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(data)
}

// JSONError is the body of every error response. It lives at package level so
// the Swagger annotations can reference it.
type JSONError struct {
	Error   bool   `json:"error" example:"true"`
	Message string `json:"message" example:"not found"`
} //@name ErrorResponse

// Response is the envelope jsonResponse wraps successful payloads in. The
// server always sends Response[any]; swagger.go declares one named
// instantiation per payload so the docs show a named schema for each.
type Response[T any] struct {
	Status    string `json:"status" example:"OK"`
	Data      T      `json:"data"`
	IsSuccess bool   `json:"is_success" example:"true"`
	Message   string `json:"message,omitempty" example:"Success"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) error {
	err := JSONError{
		Error:   true,
		Message: message,
	}
	return writeJSON(w, status, err)
}

func (app *application) jsonResponse(w http.ResponseWriter, r *http.Request, status int, data any, message string) error {
	resp := Response[any]{
		Status:    http.StatusText(status),
		Data:      data,
		IsSuccess: status >= 200 && status < 300,
		Message:   message,
	}
	if err := writeJSON(w, status, resp); err != nil {
		app.internalServerError(w, r, err)
		return err
	}
	return nil
}
