package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/jdholdren/seymour/internal/serverutil"
	"github.com/jdholdren/seymour/internal/seymour"
	"github.com/jdholdren/seymour/internal/worker"
)

type PostSubscriptionReq struct {
	FeedURL string `json:"feed_url"`
}

func validatePostSubscriptionReq(req PostSubscriptionReq) error {
	if req.FeedURL == "" {
		return seyerrs.E("feed_url is required", http.StatusBadRequest)
	}

	return nil
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

func apiFeed(f seymour.Feed) FeedResp {
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

func (s Server) postSusbcriptions(w http.ResponseWriter, r *http.Request) error {
	var (
		ctx  = r.Context()
		sess = session(r, s.secureCookie)
		body PostSubscriptionReq
	)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return seyerrs.E(err, http.StatusBadRequest)
	}
	if err := validatePostSubscriptionReq(body); err != nil {
		return err
	}

	// Start the workflow to create it and verify it
	feedID, err := worker.TriggerCreateFeedWorkflow(ctx, s.tempCli, body.FeedURL)
	if err != nil {
		// TODO: Other errors should be possible here, like a sync going bad due to a bad url
		return seyerrs.E(err, http.StatusInternalServerError)
	}
	feed, err := s.feedRepo.Feed(ctx, feedID)
	if err != nil {
		return err
	}

	// Add the feed to the user's subscriptions
	if err := s.timeline.CreateSubscription(ctx, sess.UserID, feed.ID); err != nil {
		return err
	}

	return serverutil.WriteJSON(w, http.StatusCreated, apiFeed(feed))
}

type SubscriptionResp struct {
	ID              string     `json:"id"`
	FeedID          string     `json:"feed_id"`
	CreatedAt       time.Time  `json:"created_at"`
	FeedName        string     `json:"feed_name"`
	FeedDescription string     `json:"feed_description"`
	LastSynced      *time.Time `json:"last_synced"`
}

type SubscriptionListResp struct {
	Subscriptions []SubscriptionResp `json:"subscriptions"`
}

func (s Server) getSusbcriptions(w http.ResponseWriter, r *http.Request) error {
	var (
		ctx  = r.Context()
		sess = session(r, s.secureCookie)
	)

	subs, err := s.timeline.UserSubscriptions(ctx, sess.UserID)
	if err != nil {
		return err
	}

	resp := SubscriptionListResp{
		Subscriptions: []SubscriptionResp{},
	}
	for _, sub := range subs {
		// Totally inefficient, yet sufficient:
		feed, err := s.feedRepo.Feed(ctx, sub.FeedID)
		if err != nil {
			return err
		}
		var (
			feedName        string
			feedDescription string
		)
		if feed.Title != nil {
			feedName = *feed.Title
		}
		if feed.Description != nil {
			feedDescription = *feed.Description
		}

		resp.Subscriptions = append(resp.Subscriptions, SubscriptionResp{
			ID:              sub.ID,
			FeedID:          sub.FeedID,
			CreatedAt:       sub.CreatedAt,
			FeedName:        feedName,
			FeedDescription: feedDescription,
			LastSynced:      feed.LastSyncedAt,
		})
	}
	return serverutil.WriteJSON(w, http.StatusCreated, resp)
}

type TimelineResp struct {
	Items []TimelineEntry `json:"items"`
	// TODO: Pagination details
}

type TimelineEntry struct {
	FeedName    string `json:"feed_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// TODO: Take in query params for cursor pagination
func (s Server) getUserTimeline(w http.ResponseWriter, r *http.Request) error {
	var (
		ctx     = r.Context()
		session = session(r, s.secureCookie)
		userID  = mux.Vars(r)["userID"]
	)

	// Only let the current user see their own timeline
	if session.UserID != userID {
		return seyerrs.E("not allowed", http.StatusForbidden)
	}

	tlEnts, err := s.timeline.UserTimelineEntries(ctx, userID, seymour.UserTimelineEntriesArgs{
		Status: seymour.TimelineEntryStatusApproved,
	})
	if err != nil {
		return err
	}

	feedEntIDs := make([]string, 0, len(tlEnts))
	for _, ent := range tlEnts {
		feedEntIDs = append(feedEntIDs, ent.FeedEntryID)
	}

	feedEnts, err := s.feedRepo.Entries(ctx, feedEntIDs)
	if err != nil {
		return err
	}

	feedIDs := make([]string, 0, len(feedEnts))
	for _, ent := range feedEnts {
		feedIDs = append(feedIDs, ent.FeedID)
	}

	feeds, err := s.feedRepo.Feeds(ctx, feedIDs)
	if err != nil {
		return err
	}
	// Turn into a map for fast lookup
	feedByID := make(map[string]seymour.Feed)
	for _, feed := range feeds {
		feedByID[feed.ID] = feed
	}

	resp := TimelineResp{
		Items: make([]TimelineEntry, 0, len(tlEnts)),
	}
	for _, ent := range feedEnts {
		feed := feedByID[ent.FeedID]
		resp.Items = append(resp.Items, TimelineEntry{
			FeedName:    *feed.Title,
			Title:       ent.Title,
			Description: ent.Description,
		})
	}
	return serverutil.WriteJSON(w, http.StatusOK, resp)
}
