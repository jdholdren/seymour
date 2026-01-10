package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
)

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("error encoding json response: %s", err)
	}

	return nil
}

// validator is a surface that can validate itself and return an error
// if something is wrong.
type validator interface {
	Validate() error
}

// decodeValid decodes a request and then validates it.
func decodeValid[V validator](r io.Reader) (V, error) {
	var v V
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return v, fmt.Errorf("error decoding request: %w", err)
	}
	if err := v.Validate(); err != nil {
		return v, fmt.Errorf("error validating request: %w", err)
	}

	return v, nil
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

	if err := writeJSON(w, sErr.Status, sErr); err != nil {
		slog.Error("error writing response", "error", err)
	}
}

// errRouter is a newtype around a mux router that allows attaching handlers that return errors.
type errRouter struct {
	*mux.Router
}

func (r errRouter) HandleFuncE(path string, f HandlerFuncE) *mux.Route {
	return r.Handle(path, f)
}
