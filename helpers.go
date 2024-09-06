package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any
	// problems. If len(problems) == 0 then
	// the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

// Encode writes the object to the response writer. It is usually used as the
// last step in a handler.
func Encode[T any](w http.ResponseWriter, status int, v T) {
	w.Header().Set("Content-Type", "application/json")
	// Write to buffer first to ensure the object is json encodable
	// before writing to the response writer.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error().Err(err).Msg("could not encode response")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// DecodeValid decodes the request body into the object and then validates it.
// Look at problems to see if there are any issues.
func DecodeValid[T Validator](r *http.Request) (T, map[string]string) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		problems := map[string]string{"error": fmt.Errorf("decode json: %w", err).Error()}
		return v, problems
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		problems["error"] = "invalid request"
		return v, problems
	}
	return v, nil
}
