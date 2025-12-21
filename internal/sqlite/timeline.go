package sqlite

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/jdholdren/seymour/internal/seymour"
)

const (
	subscriptionNamespace  = "sub"
	timelineEntryNamespace = "tl-entry"
)

// Usually not a fan of this pattern, but it's basically required since fx is being used.
var _ seymour.TimelineService = (*Repo)(nil)

func (r Repo) CreateSubscription(ctx context.Context, userID string, feedID string) error {
	const q = `INSERT OR IGNORE INTO subscriptions (id, user_id, feed_id) VALUES (?, ?, ?);`

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

func (r Repo) UserSubscription(ctx context.Context, userID string, feedID string) (*seymour.Subscription, error) {
	const q = `SELECT * FROM subscriptions WHERE user_id = ? AND feed_id = ?;`

	var sub seymour.Subscription
	if err := r.db.GetContext(ctx, &sub, q, userID, feedID); err != nil {
		return nil, fmt.Errorf("error selecting subscription: %s", err)
	}

	return &sub, nil
}

func (r Repo) MissingEntries(ctx context.Context, userID string) ([]seymour.MissingEntry, error) {
	const q = `
	SELECT
		fe.feed_id AS feed_id,
		fe.id AS feed_entry_id,
		subs.user_id
	FROM
		feed_entries fe
		INNER JOIN feeds ON feeds.id = fe.feed_id
		INNER JOIN subscriptions subs ON subs.feed_id = feeds.id
		INNER JOIN users ON users.id = subs.user_id
		LEFT JOIN timeline_entries ts ON ts.feed_entry_id = fe.id
		WHERE subs.user_id = ? AND ts.feed_entry_id IS NULL ;
	`

	var missingEntries []seymour.MissingEntry
	if err := r.db.SelectContext(ctx, &missingEntries, q, userID); err != nil {
		return nil, fmt.Errorf("error selecting missing entries: %s", err)
	}

	fmt.Println(missingEntries)

	return missingEntries, nil
}

func (r Repo) InsertEntry(ctx context.Context, entry seymour.TimelineEntry) error {
	const q = `INSERT OR IGNORE INTO timeline_entries (
		id,
		user_id,
		feed_entry_id,
		status,
		feed_id
	) VALUES (
		?,
		?,
		?,
		?,
		?
	);
	`

	entry.ID = fmt.Sprintf("%s-%s", uuid.New().String(), timelineEntryNamespace)
	if _, err := r.db.ExecContext(ctx, q, entry.ID, entry.UserID, entry.FeedEntryID, entry.Status, entry.FeedID); err != nil {
		return fmt.Errorf("error inserting entry: %s", err)
	}

	return nil
}

func (r Repo) EntriesNeedingJudgement(ctx context.Context, userID string) ([]seymour.TimelineEntry, error) {
	const q = `
	SELECT
		id,
		user_id,
		feed_entry_id,
		created_at,
		status
	FROM
		timeline_entries
	WHERE
		user_id = ? AND status = ?;
	`

	var entries []seymour.TimelineEntry
	if err := r.db.SelectContext(ctx, &entries, q, userID, seymour.TimelineEntryStatusRequiresJudgement); err != nil {
		return nil, fmt.Errorf("error selecting entries needing judgement: %s", err)
	}

	return entries, nil
}

func (r Repo) UpdateTimelineEntry(ctx context.Context, id string, status seymour.TimelineEntryStatus) error {
	const q = `UPDATE timeline_entries SET status = ? WHERE id = ?;`
	if _, err := r.db.ExecContext(ctx, q, status, id); err != nil {
		return fmt.Errorf("error updating entry: %s", err)
	}

	return nil
}

func (r Repo) UserTimelineEntries(ctx context.Context, userID string, args seymour.UserTimelineEntriesArgs) ([]seymour.TimelineEntry, error) {
	q := sq.Select("id", "user_id", "feed_entry_id", "created_at", "status").From("timeline_entries").OrderBy("created_at DESC")
	where := sq.Eq{
		"user_id": userID,
	}
	if args.Status != "" {
		where["status"] = args.Status
	}
	if args.Limit > 0 {
		q = q.Limit(args.Limit)
	}
	if args.FeedID != "" {
		q = q.Where("feed_id = ?", args.FeedID)
	}

	q = q.Where(where)

	query, queryArgs, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL query: %s", err)
	}

	var entries []seymour.TimelineEntry
	if err := r.db.SelectContext(ctx, &entries, query, queryArgs...); err != nil {
		return nil, fmt.Errorf("error selecting user timeline entries: %s", err)
	}

	return entries, nil
}
