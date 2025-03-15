package core

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jdholdren/seymour-agg/api"
	feedsv1 "github.com/jdholdren/seymour-agg/api/feeds/v1"
	healthv1 "github.com/jdholdren/seymour-agg/api/health/v1"
	"github.com/jdholdren/seymour-agg/internal/core/database"
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
		// Room for more
	}
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("error writing response", "err", err)
	}
}

func NewServer(cfg Config, d database.Repo) *Server {
	h := http.NewServeMux()
	attachRoutes(h, d)

	return &Server{
		Server: http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			Handler:      h,
		},
	}
}

func attachRoutes(r *http.ServeMux, d database.Repo) {
	r.HandleFunc("GET /healthz", healthz(time.Now()))
	r.HandleFunc("POST /feeds", handleCreateFeed())
}

// Creates a handler to respond to health checks.
//
// Outputs start time and build version?
func healthz(startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthv1.HealthResponse{
			UptimeSeconds: uint64(time.Since(startTime).Seconds()),
		})
	}
}

func handleCreateFeed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body feedsv1.CreateFeedRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, api.Error{
				Reason:  "invalid request",
				Message: err.Error(),
			})
			return
		}
		if err := body.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, err)
			return
		}
	}
}
