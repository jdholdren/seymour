package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jdholdren/seymour/internal/agg/model"
	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
)

const feedNamespace = "-fd"

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

func (r Repo) InsertFeed(ctx context.Context, f model.Feed) (string, error) {
	const q = `INSERT INTO feeds (id, url, title, description) VALUES (:id, :url, :title, :description);`

	f.ID = fmt.Sprintf("%s%s", uuid.New().String(), feedNamespace)
	_, err := r.db.NamedExecContext(ctx, q, f)
	if sqliteErr := (&sqlite.Error{}); errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 {
		return "", fmt.Errorf("feed already exists: %w", model.ErrConflict)
	}
	if err != nil {
		return "", fmt.Errorf("error inserting feed: %s", err)
	}

	return f.ID, nil
}

// AllFeeds retrieves _all_ feeds from the database.
func (r Repo) AllFeeds(ctx context.Context) ([]model.Feed, error) {
	const q = "SELECT * FROM feeds;"

	var feeds []model.Feed
	if err := r.db.SelectContext(ctx, &feeds, q); err != nil {
		return nil, fmt.Errorf("error selecting all feeds: %s", err)
	}

	return feeds, nil
}

func (r Repo) InsertEntries(ctx context.Context, entries []model.Entry) error {
	return errors.New("unimplemented")
}
