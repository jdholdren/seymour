package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
)

var (
	ErrConflict = errors.New("resource already exists")
	ErrNotFound = errors.New("resource not found")
)

const (
	feedNamespace  = "-fd"
	entryNamespace = "-ntry"
)

// Repo represents the surface for interacting with feeds.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates a new instance of Repo.
func NewRepo(dbx *sqlx.DB) Repo {
	return Repo{
		db: dbx,
	}
}

type (
	// Feed represents an RSS feed's details.
	Feed struct {
		ID           string     `db:"id"`
		Title        string     `db:"title"`
		URL          string     `db:"url"`
		Description  string     `db:"description"`
		LastSyncedAt *time.Time `db:"last_synced_at"`
		CreatedAt    time.Time  `db:"created_at"`
		UpdatedAt    time.Time  `db:"updated_at"`
	}

	// Entry represents a unique entry in an RSS feed.
	Entry struct {
		ID          string    `db:"id"`
		FeedID      string    `db:"feed_id"`
		GUID        string    `db:"guid"`
		Title       string    `db:"title"`
		Description string    `db:"description"`
		CreatedAt   time.Time `db:"created_at"`
	}

	// Holds the optional feeds for updating a feed.
	UpdateFeedArgs struct {
		Title       string
		Description string
		LastSynced  time.Time
	}
)

func (r Repo) Feed(ctx context.Context, id string) (Feed, error) {
	const q = `SELECT * FROM feeds WHERE id = ?;`
	var feed Feed
	err := r.db.GetContext(ctx, &feed, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Feed{}, ErrNotFound
	}
	if err != nil {
		return Feed{}, fmt.Errorf("error fetching feed: %s", err)
	}

	return feed, nil
}

func (r Repo) InsertFeed(ctx context.Context, url string) (Feed, error) {
	const q = `INSERT INTO feeds (id, url) VALUES (:id, :url);`

	f := Feed{
		ID:  fmt.Sprintf("%s%s", uuid.NewString(), feedNamespace),
		URL: url,
	}
	_, err := r.db.NamedExecContext(ctx, q, f)
	if sqliteErr := (&sqlite.Error{}); errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 {
		return Feed{}, fmt.Errorf("feed already exists: %w", ErrConflict)
	}
	if err != nil {
		return Feed{}, fmt.Errorf("error inserting feed: %s", err)
	}

	return r.Feed(ctx, f.ID)
}

// AllFeeds retrieves _all_ feeds from the database.
func (r Repo) AllFeeds(ctx context.Context) ([]Feed, error) {
	const q = "SELECT * FROM feeds;"

	var feeds []Feed
	if err := r.db.SelectContext(ctx, &feeds, q); err != nil {
		return nil, fmt.Errorf("error selecting all feeds: %s", err)
	}

	return feeds, nil
}

func (r Repo) Entry(ctx context.Context, id string) (Entry, error) {
	const q = `SELECT * FROM feed_entries WHERE id = ?;`

	var entry Entry
	err := r.db.GetContext(ctx, &entry, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("error fetching entry: %s", err)
	}

	return entry, nil
}

func (r Repo) InsertEntries(ctx context.Context, entries []Entry) error {
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

func (r Repo) UpdateFeed(ctx context.Context, id string, args UpdateFeedArgs) error {
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
