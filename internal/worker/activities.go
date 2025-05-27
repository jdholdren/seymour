package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jdholdren/seymour/internal/agg"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"go.temporal.io/sdk/temporal"
)

type activities struct {
	agg agg.Aggregator
}

// Instance to make the workflow a bit more readable
var acts = activities{}

// Fetches all RSS feeds we know about in the system.
func (a activities) AllFeeds(ctx context.Context) ([]agg.Feed, error) {
	feeds, err := a.agg.AllFeeds(ctx)
	if err != nil {
		return nil, err
	}

	return feeds, nil
}

// Goes to the url and grabs the RSS feed items.
func (a activities) SyncFeed(ctx context.Context, feedID string) error {
	return a.agg.SyncFeed(ctx, feedID)
}

func appErr(err error) error {
	if err == nil {
		return nil
	}

	seyerr := &seyerrs.Error{}
	if errors.As(err, &seyerr) {
		return temporal.NewApplicationError(seyerr.Error(), "seyerr", seyerr)
	}

	return err
}

func (a activities) CreateFeed(ctx context.Context, feedURL string) (string, error) {
	feed, err := a.agg.InsertFeed(ctx, feedURL)
	if err != nil {
		return "", fmt.Errorf("error inserting feed: %w", appErr(err))
	}

	slog.Debug("inserted feed", "feedID", feed.ID)

	return feed.ID, nil
}

func (a activities) RemoveFeed(ctx context.Context, feedID string) error {
	if err := a.agg.RemoveFeed(ctx, feedID); err != nil {
		return fmt.Errorf("error deleting feed: %w", err)
	}

	return nil
}
