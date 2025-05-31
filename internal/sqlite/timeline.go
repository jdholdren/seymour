package sqlite

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jdholdren/seymour/internal/seymour"
)

const (
	subscriptionNamespace = "sub"
)

func (r Repo) CreateSubscription(ctx context.Context, userID string, feedID string) error {
	const q = `INSERT INTO subscriptions (id, user_id, feed_id) VALUES (?, ?, ?);`

	id := fmt.Sprintf("%s-%s", uuid.New().String(), subscriptionNamespace)
	if _, err := r.db.ExecContext(ctx, q, id, userID, feedID); err != nil {
		return fmt.Errorf("error creating subscription: %w", err)
	}

	return nil
}

func (r Repo) UserSubscriptions(ctx context.Context, userID string) ([]seymour.Subscription, error) {
	const q = `SELECT * FROM subscriptions WHERE user_id = ?;`

	var subs []seymour.Subscription
	if err := r.db.SelectContext(ctx, &subs, q, userID); err != nil {
		return nil, fmt.Errorf("error selecting subscriptions: %s", err)
	}

	return subs, nil
}
