package models

import "time"

type (
	// Feed represents an RSS feed's details.
	Feed struct {
		ID          string    `db:"id"`
		Title       string    `db:"title"`
		URL         string    `db:"url"`
		Description string    `db:"description"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}

	// Entry represents a unique entry in an RSS feed.
	Entry struct{}
)
