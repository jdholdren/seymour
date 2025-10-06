package seymour

import (
	"context"
	"time"
)

type (
	TimelineService interface {
		CreateSubscription(ctx context.Context, userID string, feedID string) error
		UserSubscriptions(ctx context.Context, userID string) ([]Subscription, error)
		// Gets all of the timeline entries that users SHOULD have gotten, but haven't had inserted yet
		MissingEntries(ctx context.Context) ([]MissingEntry, error)
		// Gets a users timeline needing judgement
		EntriesNeedingJudgement(ctx context.Context, userID string) ([]TimelineEntry, error)
		InsertEntry(ctx context.Context, entry TimelineEntry) error
		UpdateTimelineEntry(ctx context.Context, id string, status TimelineEntryStatus) error
		UserTimelineEntries(ctx context.Context, userID string) ([]TimelineEntry, error)
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
)

type TimelineEntryStatus string

const (
	TimelineEntryStatusRequiresJudgement TimelineEntryStatus = "requires_judgement"
	TimelineEntryStatusApproved          TimelineEntryStatus = "approved"
	TimelineEntryStatusRejected          TimelineEntryStatus = "rejected"
)
