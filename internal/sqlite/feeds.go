package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"modernc.org/sqlite"

	"github.com/jdholdren/seymour/internal/seymour"
)

// Errors defined in the seymour package

const (
	feedNamespace  = "-fd"
	entryNamespace = "-ntry"
)

func (r Repo) Feed(ctx context.Context, id string) (seymour.Feed, error) {
	const q = `SELECT * FROM feeds WHERE id = ?;`
	var feed seymour.Feed
	err := r.db.GetContext(ctx, &feed, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return seymour.Feed{}, seymour.ErrNotFound
	}
	if err != nil {
		return seymour.Feed{}, fmt.Errorf("error fetching feed: %s", err)
	}

	return feed, nil
}

func (r Repo) Feeds(ctx context.Context, ids []string) ([]seymour.Feed, error) {
	if len(ids) == 0 {
		return []seymour.Feed{}, nil
	}

	query, args, err := sq.Select("*").From("feeds").Where(sq.Eq{"id": ids}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error constructing sql: %s", err)
	}

	var feeds []seymour.Feed
	if err := r.db.SelectContext(ctx, &feeds, query, args...); err != nil {
		return nil, fmt.Errorf("error fetching feeds: %s", err)
	}

	return feeds, nil
}

func (r Repo) FeedByURL(ctx context.Context, url string) (seymour.Feed, error) {
	const q = `SELECT * FROM feeds WHERE url = ?;`

	var feed seymour.Feed
	err := r.db.GetContext(ctx, &feed, q, url)
	if errors.Is(err, sql.ErrNoRows) {
		return seymour.Feed{}, seymour.ErrNotFound
	}
	if err != nil {
		return seymour.Feed{}, fmt.Errorf("error fetching feed: %s", err)
	}

	return feed, nil
}

func (r Repo) InsertFeed(ctx context.Context, url string) (seymour.Feed, error) {
	const q = `INSERT INTO feeds (id, url) VALUES (:id, :url);`
	f := seymour.Feed{
		ID:  fmt.Sprintf("%s%s", uuid.NewString(), feedNamespace),
		URL: url,
	}
	_, err := r.db.NamedExecContext(ctx, q, f)
	if sqliteErr := (&sqlite.Error{}); errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 {
		return seymour.Feed{}, fmt.Errorf("feed already exists: %w", seymour.ErrConflict)
	}
	if err != nil {
		return seymour.Feed{}, fmt.Errorf("error inserting feed: %s", err)
	}

	return r.Feed(ctx, f.ID)
}

func (r Repo) DeleteFeed(ctx context.Context, id string) error {
	const q = `DELETE FROM feeds WHERE id = ?;`

	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("error deleting feed: %s", err)
	}

	return nil
}

// AllFeeds retrieves _all_ feeds from the database.
func (r Repo) AllFeeds(ctx context.Context) ([]seymour.Feed, error) {
	const q = "SELECT * FROM feeds;"

	var feeds []seymour.Feed
	if err := r.db.SelectContext(ctx, &feeds, q); err != nil {
		return nil, fmt.Errorf("error selecting all feeds: %s", err)
	}

	return feeds, nil
}

func (r Repo) Entry(ctx context.Context, id string) (seymour.FeedEntry, error) {
	const q = `SELECT * FROM feed_entries WHERE id = ?;`

	var entry seymour.FeedEntry
	err := r.db.GetContext(ctx, &entry, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return seymour.FeedEntry{}, seymour.ErrNotFound
	}
	if err != nil {
		return seymour.FeedEntry{}, fmt.Errorf("error fetching entry: %s", err)
	}

	return entry, nil
}

func (r Repo) Entries(ctx context.Context, ids []string) ([]seymour.FeedEntry, error) {
	if len(ids) == 0 {
		return []seymour.FeedEntry{}, nil
	}

	query, args, err := sq.Select("*").From("feed_entries").Where(sq.Eq{"id": ids}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error constructing sql: %s", err)
	}

	var entries []seymour.FeedEntry
	if err := r.db.SelectContext(ctx, &entries, query, args...); err != nil {
		return nil, fmt.Errorf("error fetching entries: %s", err)
	}

	return entries, nil
}

func (r Repo) InsertEntries(ctx context.Context, entries []seymour.FeedEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Create id's for the entries
	for i := range entries {
		entries[i].ID = fmt.Sprintf("%s%s", uuid.New().String(), entryNamespace)
	}

	const q = `INSERT INTO feed_entries (id, feed_id, title, description, guid)
	VALUES (:id, :feed_id, :title, :description, :guid)
	ON CONFLICT(guid) DO NOTHING;`
	if _, err := r.db.NamedExecContext(ctx, q, entries); err != nil {
		return fmt.Errorf("error inserting entries; %s", err)
	}

	return nil
}

func (r Repo) UpdateFeed(ctx context.Context, id string, args seymour.UpdateFeedArgs) error {
	q := sq.Update("feeds")
	if args.Title != "" {
		q = q.Set("title", args.Title)
	}
	if args.Description != "" {
		q = q.Set("description", args.Description)
	}
	if !args.LastSynced.IsZero() {
		q = q.Set("last_synced_at", args.LastSynced)
	}
	q = q.Where(sq.Eq{"id": id})

	query, qArgs, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("error constructing sql: %s", err)
	}
	if _, err := r.db.ExecContext(ctx, query, qArgs...); err != nil {
		return fmt.Errorf("error executing feed update")
	}

	return nil
}
