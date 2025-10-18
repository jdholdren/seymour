package seymour

import (
	"context"
	"time"
)

type (
	FeedService interface {
		Feed(ctx context.Context, id string) (Feed, error)
		Feeds(ctx context.Context, ids []string) ([]Feed, error)
		FeedByURL(ctx context.Context, url string) (Feed, error)
		InsertFeed(ctx context.Context, url string) (Feed, error)
		DeleteFeed(ctx context.Context, id string) error
		AllFeeds(ctx context.Context) ([]Feed, error)
		Entry(ctx context.Context, id string) (FeedEntry, error)
		Entries(ctx context.Context, ids []string) ([]FeedEntry, error)
		InsertEntries(ctx context.Context, entries []FeedEntry) error
		UpdateFeed(ctx context.Context, id string, args UpdateFeedArgs) error
	}

	// Feed represents an RSS feed's details.
	Feed struct {
		ID           string     `db:"id"`
		Title        *string    `db:"title"`
		URL          string     `db:"url"`
		Description  *string    `db:"description"`
		LastSyncedAt *time.Time `db:"last_synced_at"`
		CreatedAt    time.Time  `db:"created_at"`
		UpdatedAt    time.Time  `db:"updated_at"`
	}

	// FeedEntry represents a unique entry in an RSS feed.
	FeedEntry struct {
		ID          string     `db:"id"`
		FeedID      string     `db:"feed_id"`
		GUID        string     `db:"guid"`
		Title       string     `db:"title"`
		Description string     `db:"description"`
		CreatedAt   time.Time  `db:"created_at"`
		PublishTime *time.Time `db:"publish_time"`
	}

	// Holds the optional fields for updating a feed.
	UpdateFeedArgs struct {
		Title       string
		Description string
		LastSynced  time.Time
	}
)
