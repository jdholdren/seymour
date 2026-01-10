package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/jdholdren/seymour/internal/seymour"
)

// Viewer is the structured data about the current user in the frontend.
type (
	Viewer struct {
		UserID    string    `json:"user_id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		Prompt    string    `json:"prompt"`

		// Information about the user's nav bar
		PersonalSubscriptions map[string]ViewerSubscription `json:"subscriptions"`
	}

	ViewerSubscription struct {
		Name        string `json:"name"`
		FeedID      string `json:"feed_id"`
		Description string `json:"description"`
	}
)

func (s Server) handleViewer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	sess := session(r, s.secureCookie)
	if sess.UserID == "" {
		return writeJSON(w, http.StatusOK, struct{}{})
	}
	usr, err := s.repo.User(r.Context(), sess.UserID)
	if errors.Is(err, seymour.ErrNotFound) {
		return writeJSON(w, http.StatusOK, struct{}{})
	}
	if err != nil {
		return err
	}

	// Get the feeds that the user has subscribed to.
	// This will populate the nav bar with the individual filters for their personal timeline.
	subs, err := s.repo.UserSubscriptions(ctx, usr.ID)
	if err != nil {
		return err
	}

	// Get the feed names themselves.
	var feedIDs []string
	for _, sub := range subs {
		feedIDs = append(feedIDs, sub.FeedID)
	}
	feeds, err := s.repo.Feeds(ctx, feedIDs)
	if err != nil {
		return err
	}
	feedsByID := make(map[string]seymour.Feed)
	for _, feed := range feeds {
		feedsByID[feed.ID] = feed
	}

	viewerSubs := make(map[string]ViewerSubscription)
	for _, feed := range feeds {
		feed := feedsByID[feed.ID]
		var title, desc string
		if feed.Title != nil {
			title = *feed.Title
		}
		if feed.Description != nil {
			desc = *feed.Description
		}

		viewerSubs[feed.ID] = ViewerSubscription{
			Name:        title,
			FeedID:      feed.ID,
			Description: desc,
		}
	}

	return writeJSON(w, http.StatusOK, Viewer{
		UserID:                usr.ID,
		Email:                 usr.Email,
		CreatedAt:             usr.CreatedAt,
		Prompt:                usr.Prompt,
		PersonalSubscriptions: viewerSubs,
	})
}
