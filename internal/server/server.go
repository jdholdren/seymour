package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type (
	// Server is the HTTP portion serving the public API.
	Server struct {
		http.Server
	}

	// Config holds all of the different options for making a
	// server.
	Config struct {
		Port int
	}
)

func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("error encoding json response: %s", err)
	}

	return nil
}

// Validator is a surface that can validate itself and return an error
// if something is wrong.
type Validator interface {
	Validate() error
}

// DecodeValid decodes a request and then validates it.
func DecodeValid[V Validator](r io.Reader) (V, error) {
	var v V
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return v, fmt.Errorf("error decoding request: %w", err)
	}
	if err := v.Validate(); err != nil {
		return v, fmt.Errorf("error validating request: %w", err)
	}

	return v, nil
}

func NewServer(port int) (*Server, *http.ServeMux) {
	m := http.NewServeMux()

	// TOOD: Do other useful stuff with a base server
	return &Server{
		Server: http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			Handler:      accessLogWrapper{inner: m},
		},
	}, m
}

// Implements [http.Handler] to wrap each call with an access log.
type accessLogWrapper struct {
	inner http.Handler
}

func (alm accessLogWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("request received", "method", r.Method, "path", r.URL.Path)
	start := time.Now()

	alm.inner.ServeHTTP(w, r)

	slog.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
}

// HandlerFuncE is a modified type of [http.HandlerFunc] that returns an error.
type HandlerFuncE func(w http.ResponseWriter, r *http.Request) error

func (f HandlerFuncE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := f(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
