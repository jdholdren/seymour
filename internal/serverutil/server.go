package serverutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
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

func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request received", "method", r.Method, "path", r.URL.Path)
		start := time.Now()

		writer := &respCodeWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)

		slog.Info("request completed",
			"method", r.Method,
			"url", r.URL.String(),
			"duration", time.Since(start),
			"status_code", writer.code,
		)
	})
}

// To trap the response status code for logging later.
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
		slog.Info("no error coming back")
		return
	}

	// Either it's already a structured error, or coerce it to one
	sErr := &seyerrs.Error{}
	if !errors.As(err, &sErr) {
		slog.Error("non seyerr", "err", err)
		sErr = seyerrs.E(http.StatusInternalServerError, "internal server error")
	}

	if err := WriteJSON(w, sErr.Status, sErr); err != nil {
		slog.Error("error writing response", "error", err)
	}
}

// ErrRouter is a newtype around a mux router that allows attaching handlers that return errors.
type ErrRouter struct {
	*mux.Router
}

func (r ErrRouter) HandleFuncE(path string, f HandlerFuncE) *mux.Route {
	return r.Handle(path, f)
}
