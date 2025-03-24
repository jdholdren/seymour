package model

import (
	"errors"
	"time"
)

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
		ID          string `db:"id"`
		FeedID      string `db:"feed_id"`
		GUID        string `db:"guid"`
		Title       string `db:"title"`
		Description string `db:"description"`
	}
)

var (
	ErrConflict = errors.New("conflict")
)
