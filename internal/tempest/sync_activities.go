package tempest

import (
	"context"
	"log/slog"

	"github.com/jdholdren/seymour/internal/agg"
)

// SyncRSSFeed grabs all entries and updates the feed's metadata.
func (a activities) SyncRSSFeed(ctx context.Context, feedID string) error {
	slog.Info("syncing feed", "feed_id", feedID)

	if err := a.agg.SyncFeed(ctx, feedID); err != nil {
		return err
	}

	return nil
}

// Fetches all RSS feeds we know about in the system.
func (a activities) AllRSSFeeds(ctx context.Context) ([]agg.Feed, error) {
	feeds, err := a.agg.Feeds(ctx)
	if err != nil {
		return nil, err
	}

	return feeds, nil
}
