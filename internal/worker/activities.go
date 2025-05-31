package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jdholdren/seymour/internal/seymour"
	"github.com/jdholdren/seymour/internal/sync"
)

type activities struct {
	feedRepo seymour.FeedRepo
}

// Instance to make the workflow a bit more readable
var acts = activities{}

// Fetches all RSS feeds we know about in the system.
func (a activities) AllFeeds(ctx context.Context) ([]seymour.Feed, error) {
	feeds, err := a.feedRepo.AllFeeds(ctx)
	if err != nil {
		return nil, err
	}

	return feeds, nil
}

// Fetches a single feed
func (a activities) Feed(ctx context.Context, feedID string) (seymour.Feed, error) {
	feed, err := a.feedRepo.Feed(ctx, feedID)
	if err != nil {
		return seymour.Feed{}, err
	}

	return feed, nil
}

// Goes to the url and grabs the RSS feed items.
func (a activities) SyncFeed(ctx context.Context, feedID string) error {
	feed, err := a.feedRepo.Feed(ctx, feedID)
	if err != nil {
		return err
	}

	feed, entries, err := sync.Feed(ctx, feed.ID, feed.URL)
	if err != nil {
		return err
	}

	if err := a.feedRepo.UpdateFeed(ctx, feed.ID, seymour.UpdateFeedArgs{
		Title:       *feed.Title,
		Description: *feed.Title,
		LastSynced:  time.Now(),
	}); err != nil {
		return err
	}
	if err := a.feedRepo.InsertEntries(ctx, entries); err != nil {
		return err
	}

	return err
}

func (a activities) CreateFeed(ctx context.Context, feedURL string) (string, error) {
	feed, err := a.feedRepo.InsertFeed(ctx, feedURL)
	if errors.Is(err, seymour.ErrConflict) {
		// Fetch the feed from the database
		feed, err = a.feedRepo.FeedByURL(ctx, feedURL)
		if err != nil {
			return "", fmt.Errorf("error fetching conflicting feed: %s", err)
		}

		return feed.ID, nil
	}
	if err != nil {
		return "", fmt.Errorf("error inserting feed: %w", err)
	}

	slog.Debug("inserted feed", "feedID", feed.ID)

	return feed.ID, nil
}

func (a activities) RemoveFeed(ctx context.Context, feedID string) error {
	if err := a.feedRepo.DeleteFeed(ctx, feedID); err != nil {
		return fmt.Errorf("error deleting feed: %w", err)
	}

	return nil
}
