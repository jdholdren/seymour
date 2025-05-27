// Package agg provides the aggregation daemon that scrapes feeds that
// it has been configured to.
package agg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"go.temporal.io/sdk/client"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/agg/db"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/jdholdren/seymour/internal/server"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		server.Server

		repo        db.Repo
		temporalCli client.Client
	}

	ServerConfig struct {
		Port int
	}

	Params struct {
		fx.In

		Config      ServerConfig
		Repo        db.Repo
		TemporalCli client.Client
	}
)

func NewServer(lc fx.Lifecycle, p Params) Server {
	srvr := Server{
		Server:      server.NewServer(fmt.Sprintf(":%d", p.Config.Port)),
		repo:        p.Repo,
		temporalCli: p.TemporalCli,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srvr.ListenAndServe()

			slog.Debug("started agg server", "port", p.Config.Port)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srvr.Shutdown(ctx)
		},
	})

	srvr.HandleFuncE("/subscriptions", srvr.handleCreateFeed).Methods(http.MethodPost)

	return srvr
}

type CreateFeedReq struct {
	FeedURL string `json:"feed_url"`
}

func (s Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) error {
	var (
		ctx  = r.Context()
		body CreateFeedReq
	)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return seyerrs.E(err, http.StatusBadRequest)
	}

	feedID, err := workflows{}.TriggerCreateFeedWorkflow(ctx, s.temporalCli, body.FeedURL)
	if err != nil {
		return seyerrs.E(err)
	}

	feed, err := s.repo.Feed(ctx, feedID)
	if err != nil {
		return seyerrs.E(err)
	}

	return server.WriteJSON(w, http.StatusCreated, feed)
}
