// Package agg provides the aggregation daemon that scrapes feeds that
// it has been configured to.
package agg

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"go.uber.org/fx"

	feedsv1 "github.com/jdholdren/seymour/api/feeds/v1"
	seyerrs "github.com/jdholdren/seymour/errors"
	"github.com/jdholdren/seymour/internal/agg/database"
	"github.com/jdholdren/seymour/internal/agg/model"
	"github.com/jdholdren/seymour/internal/server"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		*server.Server

		repo   database.Repo
		syncer *Syncer
	}

	Config struct {
		Port int
	}

	Params struct {
		fx.In

		Config Config
		Repo   database.Repo
		Syncer *Syncer
	}
)

func NewServer(lc fx.Lifecycle, p Params) Server {
	var (
		s, r = server.NewServer(p.Config.Port)
	)
	srvr := Server{
		Server: s,
		repo:   p.Repo,
		syncer: p.Syncer,
	}

	// Attach routes
	r.Handle("POST /v1/feeds", server.HandlerFuncE(srvr.handleCreateFeed))
	r.Handle("GET /v1/feeds/{id}", server.HandlerFuncE(srvr.handleGetFeed))
	r.Handle("GET /v1/entries/{id}", server.HandlerFuncE(srvr.handleGetEntry))

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srvr.ListenAndServe()

			slog.Debug("started aggregation server", "port", p.Config.Port)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srvr.Shutdown(ctx)
		},
	})

	return srvr
}

func (s Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) error {
	body, err := server.DecodeValid[feedsv1.CreateFeedRequest](r.Body)
	if err != nil {
		return seyerrs.E(err, http.StatusBadRequest)
	}

	feed, err := s.repo.InsertFeed(r.Context(), model.Feed{
		URL: body.URL,
	})
	if errors.Is(err, model.ErrConflict) {
		return seyerrs.E(err, http.StatusConflict)
	}
	if err != nil {
		return err
	}

	// Make sure that its added to the syncer
	s.syncer.AddFeed(feed)

	resp := feedsv1.CreateFeedResponse{
		ID: feed.ID,
	}
	return server.WriteJSON(w, http.StatusCreated, resp)
}

func (s Server) handleGetFeed(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	feed, err := s.repo.Feed(r.Context(), id)
	if errors.Is(err, model.ErrNotFound) {
		return seyerrs.E(err, http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	resp := feedsv1.Feed{
		ID:           feed.ID,
		Title:        feed.Title,
		Description:  feed.Description,
		LastSyncedAt: feed.LastSyncedAt,
		CreatedAt:    feed.CreatedAt,
		UpdatedAt:    feed.UpdatedAt,
	}
	return server.WriteJSON(w, http.StatusOK, resp)
}

func (s Server) handleGetEntry(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	entry, err := s.repo.Entry(r.Context(), id)
	if errors.Is(err, model.ErrNotFound) {
		return seyerrs.E(err, http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	resp := feedsv1.Entry{
		ID:          entry.ID,
		Title:       entry.Title,
		Description: entry.Description,
		FeedID:      entry.FeedID,
		GUID:        entry.GUID,
	}

	return server.WriteJSON(w, http.StatusOK, resp)
}
