package timeline

import "time"

type (
	// Subscription is a user's desire to see posts from a given feed in their timeline.
	Subscription struct {
		ID        string    `db:"id"`
		UserID    string    `db:"user_id"`    // The user the subscription belongs to.
		FeedID    string    `db:"feed_id"`    // The feed the subscription is for.
		CreatedAt time.Time `db:"created_at"` // The time the subscription was created.
		UpdatedAt time.Time `db:"updated_at"` // The time the subscription was last updated.
	}

	Post struct {
		ID        string    `db:"id"`
		UserID    string    `db:"user_id"`    // The user the post belongs to.
		FeedID    string    `db:"feed_id"`    // The feed the post belongs to.
		EntryID   string    `db:"entry_id"`   // The entry the post is for.
		CreatedAt time.Time `db:"created_at"` // The time the post was inserted.
		UpdatedAt time.Time `db:"updated_at"` // The time the post was last updated.
	}
)
