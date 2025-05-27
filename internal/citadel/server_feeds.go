package citadel

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jdholdren/seymour/internal/agg"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/jdholdren/seymour/internal/server"
	"github.com/jdholdren/seymour/internal/worker"
)

type PostSubscriptionReq struct {
	FeedURL string `json:"feed_url"`
}

type FeedResp struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Description  string     `json:"description"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func apiFeed(f agg.Feed) FeedResp {
	var (
		title string
		desc  string
	)
	if f.Title != nil {
		title = *f.Title
	}
	if f.Description != nil {
		desc = *f.Description
	}

	return FeedResp{
		ID:           f.ID,
		Title:        title,
		URL:          f.URL,
		Description:  desc,
		LastSyncedAt: f.LastSyncedAt,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}

func (s Server) postSusbcription(w http.ResponseWriter, r *http.Request) error {
	var (
		ctx  = r.Context()
		body PostSubscriptionReq
	)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return seyerrs.E(err, http.StatusBadRequest)
	}

	// Start the workflow to create it and verify it
	feedID, err := worker.TriggerCreateFeedWorkflow(ctx, s.tempCli, body.FeedURL)
	if err != nil {
		// TODO: Other errors should be possible here, like a sync going bad due to a bad url
		return seyerrs.E(err, http.StatusInternalServerError)
	}
	feed, err := s.aggregator.Feed(ctx, feedID)
	if err != nil {
		return err
	}

	// TODO: Add the feed to the user's subscriptions

	return server.WriteJSON(w, http.StatusCreated, apiFeed(feed))
}
