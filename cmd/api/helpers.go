package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"runtime"
)

// writeJSON writes aribtrary data out as JSON
func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for k, v := range headers[0] {
			w.Header()[k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(out)

	return nil
}

// readJSON reads json from request body into data. We only accept a single json value in the body
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1048576 // max one megabyte in request body
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	// we only allow one entry in the json file
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only have a single JSON value")
	}

	return nil
}

// badRequest sends a JSON response with status http.StatusBadRequest, describing the error
func (app *application) badRequest(w http.ResponseWriter, r *http.Request, err error) error {
	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	payload.Error = true
	payload.Message = err.Error()

	out, err := json.MarshalIndent(payload, "", "\t")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write(out)
	return nil
}

func (app *application) invalidCredentials(w http.ResponseWriter) error {
	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	payload.Error = true
	payload.Message = "invalid authentication credentials"

	err := app.writeJSON(w, http.StatusUnauthorized, payload)
	if err != nil {
		return err
	}
	return nil
}

func (app *application) passwordMatches(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func (app *application) failedValidation(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	var payload struct {
		Error   bool              `json:"error"`
		Message string            `json:"message"`
		Errors  map[string]string `json:"errors"`
	}

	payload.Error = true
	payload.Message = "failed validation"
	payload.Errors = errors
	app.writeJSON(w, http.StatusUnprocessableEntity, payload)
}

func trace() {
	const trace = true // (true or false)
	if trace {
		pc := make([]uintptr, 15)
		n := runtime.Callers(2, pc)
		frames := runtime.CallersFrames(pc[:n])
		frame, _ := frames.Next()
		fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
	}
}
