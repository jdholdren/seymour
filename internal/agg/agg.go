// Package agg provides the aggregation daemon that scrapes feeds that
// it has been configured to.
package agg

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/fx"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
)

type (
	// Feed represents an RSS feed's details.
	Feed struct {
		ID           string     `db:"id"`
		Title        string     `db:"title"`
		URL          string     `db:"url"`
		Description  string     `db:"description"`
		LastSyncedAt *time.Time `db:"last_synced_at"`
		CreatedAt    time.Time  `db:"created_at"`
		UpdatedAt    time.Time  `db:"updated_at"`
	}

	// Entry represents a unique entry in an RSS feed.
	Entry struct {
		ID          string    `db:"id"`
		FeedID      string    `db:"feed_id"`
		GUID        string    `db:"guid"`
		Title       string    `db:"title"`
		Description string    `db:"description"`
		CreatedAt   time.Time `db:"created_at"`
	}

	// Holds the optional feeds for updating a feed.
	UpdateFeedArgs struct {
		Title       string
		Description string
		LastSynced  time.Time
	}
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Service struct {
		repo Repo
	}

	Params struct {
		fx.In

		Repo Repo
	}
)

func NewService(p Params) Service {
	return Service{
		repo: p.Repo,
	}
}

func (s Service) CreateFeed(ctx context.Context, url string) (Feed, error) {
	feed, err := s.repo.InsertFeed(ctx, url)
	if errors.Is(err, errConflict) {
		return Feed{}, seyerrs.E(err, http.StatusConflict)
	}
	if err != nil {
		return Feed{}, seyerrs.E(err)
	}

	// TODO: Enqueue a job to sync the feed

	return feed, nil
}

func (s Service) Feeds(ctx context.Context) ([]Feed, error) {
	feeds, err := s.repo.AllFeeds(ctx)
	if err != nil {
		return nil, seyerrs.E(err)
	}

	return feeds, nil
}
