package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
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

func NewServer(name string, port int) (*Server, *http.ServeMux) {
	m := http.NewServeMux()

	return &Server{
		Server: http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			Handler:      accessLogWrapper{serverName: name, inner: m},
		},
	}, m
}

// Implements [http.Handler] to wrap each call with an access log.
type accessLogWrapper struct {
	serverName string
	inner      http.Handler
}

func (alw accessLogWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("request received", "server", alw.serverName, "method", r.Method, "path", r.URL.Path)
	start := time.Now()

	writer := &respCodeWriter{ResponseWriter: w}
	alw.inner.ServeHTTP(writer, r)

	slog.Info("request completed",
		"server", alw.serverName,
		"method", r.Method,
		"url", r.URL.String(),
		"duration", time.Since(start),
		"status_code", writer.code,
	)
}

type respCodeWriter struct {
	http.ResponseWriter
	code int
}

func (w *respCodeWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// HandlerFuncE is a modified type of [http.HandlerFunc] that returns an error.
type HandlerFuncE func(w http.ResponseWriter, r *http.Request) error

func (f HandlerFuncE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := f(w, r)
	if err == nil {
		return
	}

	// Either it's already a structured error, or coerce it to one
	sErr := &seyerrs.Error{}
	if !errors.As(err, &sErr) {
		sErr = seyerrs.E(http.StatusInternalServerError, err)
	}

	WriteJSON(w, sErr.Status, sErr)
}
