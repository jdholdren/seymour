// Package agg provides the aggregation daemon that scrapes feeds that
// it has been configured to.
package agg

import (
	"net/http"

	"github.com/jdholdren/seymour/api"
	feedsv1 "github.com/jdholdren/seymour/api/feeds/v1"
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
)

func NewServer(port int, repo database.Repo, syncer *Syncer) Server {
	var (
		s, r = server.NewServer(port)
	)
	srvr := Server{
		Server: s,
		repo:   repo,
		syncer: syncer,
	}

	// Attach routes
	r.HandleFunc("POST /v1/feeds", srvr.handleCreateFeed)

	return srvr
}

func (s Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	body, err := server.DecodeValid[feedsv1.CreateFeedRequest](r.Body)
	if err != nil {
		server.WriteJSON(w, http.StatusBadRequest, api.Error{
			Reason:  "invalid request",
			Message: err.Error(),
		})
		return
	}

	feed, err := s.repo.InsertFeed(r.Context(), model.Feed{
		URL: body.URL,
	})
	if err != nil {
		server.WriteJSON(w, http.StatusInternalServerError, api.Error{
			Reason:  "internal error", // TODO: Make these consts
			Message: err.Error(),
		})
		return
	}

	// Make sure that its added to the syncer
	s.syncer.AddFeed(feed)

	resp := feedsv1.CreateFeedResponse{
		ID: feed.ID,
	}
	server.WriteJSON(w, http.StatusCreated, resp)
}
