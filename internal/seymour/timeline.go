package seymour

import (
	"context"
	"time"
)

type (
	Subscription struct {
		ID        string    `db:"id"`
		UserID    string    `db:"user_id"`
		FeedID    string    `db:"feed_id"`
		CreatedAt time.Time `db:"created_at"`
	}

	TimelineEntry struct {
		ID             string    `db:"id"`
		SubscriptionID string    `db:"subscription_id"`
		UserID         string    `db:"user_id"`
		FeedID         string    `db:"feed_id"`
		CreatedAt      time.Time `db:"created_at"`

		// For curation: if the entry has been approved or not by the AI
		Approved bool `db:"approved"`
	}

	TimelineRepo interface {
		CreateSubscription(ctx context.Context, userID string, feedID string) error
		UserSubscriptions(ctx context.Context, userID string) ([]Subscription, error)
	}
)
