// Package timeline provides the service that manages a user's timeline.
// It manages their subscriptions and builds their timeline according to preferences.
package timeline

import (
	"context"
	"log/slog"
	"net/http"

	"go.uber.org/fx"

	v1 "github.com/jdholdren/seymour/api/timeline/v1"
	"github.com/jdholdren/seymour/internal/agg/database"
	"github.com/jdholdren/seymour/internal/server"
)

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
	s, r := server.NewServer("timeline", p.Config.Port)
	srvr := Server{
		Server: s,
	}

	r.Handle("POST /v1/users/{id}/subscriptions", server.HandlerFuncE(srvr.handleCreateSubscription))

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
	_, err := server.DecodeValid[v1.CreateSubscriptionRequest](r.Body)
	if err != nil {
		return err
	}

	// TODO: Implement
	return nil
}
