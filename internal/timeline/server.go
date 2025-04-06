package timeline

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jdholdren/seymour/internal/agg/database"
	"github.com/jdholdren/seymour/internal/server"
	"go.uber.org/fx"
)

// Package timeline provides the service that manages a user's timeline.
// It manages their subscriptions and builds their timeline according to preferences.

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		*server.Server
		repo database.Repo
	}

	Config struct {
		Port    int
		AggHost string
	}

	Params struct {
		fx.In

		Config Config
	}
)

func NewServer(lc fx.Lifecycle, p Params) Server {
	s, _ := server.NewServer(p.Config.Port)
	srvr := Server{
		Server: s,
	}

	// TODO: Attach routes

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srvr.ListenAndServe()

			slog.Debug("started timeline server", "port", p.Config.Port)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srvr.Shutdown(ctx)
		},
	})

	return srvr

}

// Attaches a subscription to the user's timeline.
func (s Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) error {
	// TODO: Implement
	return nil
}
