package seymour

import (
	"context"
	"errors"
	"time"
)

var (
	ErrConflict = errors.New("resource already exists")
	ErrNotFound = errors.New("resource not found")
)

type (
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
		ID          string    `db:"id"`
		FeedID      string    `db:"feed_id"`
		GUID        string    `db:"guid"`
		Title       string    `db:"title"`
		Description string    `db:"description"`
		CreatedAt   time.Time `db:"created_at"`
	}

	FeedService interface {
		Feed(ctx context.Context, id string) (Feed, error)
		FeedByURL(ctx context.Context, url string) (Feed, error)
		InsertFeed(ctx context.Context, url string) (Feed, error)
		DeleteFeed(ctx context.Context, id string) error
		AllFeeds(ctx context.Context) ([]Feed, error)
		Entry(ctx context.Context, id string) (FeedEntry, error)
		InsertEntries(ctx context.Context, entries []FeedEntry) error
		UpdateFeed(ctx context.Context, id string, args UpdateFeedArgs) error
	}

	// Holds the optional feeds for updating a feed.
	UpdateFeedArgs struct {
		Title       string
		Description string
		LastSynced  time.Time
	}

	Subscription struct {
		ID        string    `db:"id"`
		UserID    string    `db:"user_id"`
		FeedID    string    `db:"feed_id"`
		CreatedAt time.Time `db:"created_at"`
	}

	TimelineEntry struct {
		ID          string    `db:"id"`
		UserID      string    `db:"user_id"`
		FeedEntryID string    `db:"feed_entry_id"`
		CreatedAt   time.Time `db:"created_at"`

		// For curation: if the entry has been approved or not by the AI
		Status TimelineEntryStatus `db:"status"`
	}

	// MissingEntry is an instance where a user should have gotten the feed entry put into their timeline.
	MissingEntry struct {
		FeedEntryID string `db:"feed_entry_id"`
		UserID      string `db:"user_id"`
	}

	TimelineEntryWithFeed struct {
		FeedEntry

		TimelineEntryID string `db:"timeline_entry_id"`
	}

	TimelineService interface {
		CreateSubscription(ctx context.Context, userID string, feedID string) error
		UserSubscriptions(ctx context.Context, userID string) ([]Subscription, error)
		// Gets all of the timeline entries that users SHOULD have gotten, but haven't had inserted yet
		MissingEntries(ctx context.Context) ([]MissingEntry, error)
		// Gets a users timeline needing judgement
		EntriesNeedingJudgement(ctx context.Context, userID string) ([]TimelineEntryWithFeed, error)
		InsertEntry(ctx context.Context, entry TimelineEntry) error
		UpdateTimelineEntry(ctx context.Context, id string, status TimelineEntryStatus) error
	}
)

type TimelineEntryStatus string

const (
	TimelineEntryStatusRequiresJudgement TimelineEntryStatus = "requires_judgement"
	TimelineEntryStatusApproved          TimelineEntryStatus = "approved"
	TimelineEntryStatusRejected          TimelineEntryStatus = "rejected"
)
