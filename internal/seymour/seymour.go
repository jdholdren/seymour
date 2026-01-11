package seymour

import (
	"context"
	"time"
)

// Repository provides a unified interface for all data operations.
// It combines the functionality of FeedService, TimelineService, and UserService
// into a single interface to simplify dependency injection and testing.
type Repository interface {
	// Feed operations
	Feed(ctx context.Context, id string) (Feed, error)
	Feeds(ctx context.Context, ids []string) ([]Feed, error)
	FeedByURL(ctx context.Context, url string) (Feed, error)
	InsertFeed(ctx context.Context, url string) (Feed, error)
	DeleteFeed(ctx context.Context, id string) error
	CountAllFeeds(ctx context.Context) (int, error)
	FeedIDs(ctx context.Context, offset, pageSize int) ([]string, error)
	Entry(ctx context.Context, id string) (FeedEntry, error)
	Entries(ctx context.Context, ids []string) ([]FeedEntry, error)
	InsertEntries(ctx context.Context, entries []FeedEntry) error
	UpdateFeed(ctx context.Context, id string, args UpdateFeedArgs) error

	// Timeline operations
	CreateSubscription(ctx context.Context, userID string, feedID string) error
	UserSubscriptions(ctx context.Context, userID string) ([]Subscription, error)
	MissingEntries(ctx context.Context, userID string) ([]MissingEntry, error)
	EntriesNeedingJudgement(ctx context.Context, userID string, limit uint) ([]TimelineEntry, error)
	InsertEntry(ctx context.Context, entry TimelineEntry) error
	UpdateTimelineEntry(ctx context.Context, id string, status TimelineEntryStatus) error
	UserTimelineEntries(ctx context.Context, userID string, args UserTimelineEntriesArgs) ([]TimelineEntry, error)
	UpdateUserPrompt(ctx context.Context, userID string, prompt string) error

	// User operations
	EnsureUser(ctx context.Context, usr User) (User, error)
	User(ctx context.Context, id string) (User, error)
	UserByGithubID(ctx context.Context, githubID string) (User, error)
	AllUserIDs(ctx context.Context) ([]string, error)
}

// Feed represents an RSS feed's details.
type Feed struct {
	ID           string     `db:"id"`
	Title        *string    `db:"title"`
	URL          string     `db:"url"`
	Description  *string    `db:"description"`
	LastSyncedAt *time.Time `db:"last_synced_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// FeedEntry represents a unique entry in an RSS feed.
type FeedEntry struct {
	ID          string     `db:"id"`
	FeedID      string     `db:"feed_id"`
	GUID        string     `db:"guid"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	CreatedAt   time.Time  `db:"created_at"`
	PublishTime *time.Time `db:"publish_time"`
	Link        string     `db:"link"`
}

// UpdateFeedArgs holds the optional fields for updating a feed.
type UpdateFeedArgs struct {
	Title       string
	Description string
	LastSynced  time.Time
}

// User represents a user in the system.
type User struct {
	ID        string    `db:"id"`
	GithubID  string    `db:"github_id"`
	Email     string    `db:"email"`
	Prompt    string    `db:"prompt"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Subscription represents a user's subscription to a feed.
type Subscription struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	FeedID    string    `db:"feed_id"`
	CreatedAt time.Time `db:"created_at"`
}

// TimelineEntry represents an entry in a user's timeline.
type TimelineEntry struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	FeedEntryID string    `db:"feed_entry_id"`
	CreatedAt   time.Time `db:"created_at"`
	FeedID      string    `db:"feed_id"`

	// For curation: if the entry has been approved or not by the AI
	Status TimelineEntryStatus `db:"status"`
}

// MissingEntry is an instance where a user should have gotten the feed entry put into their timeline.
type MissingEntry struct {
	FeedEntryID string `db:"feed_entry_id"`
	FeedID      string `db:"feed_id"`
	UserID      string `db:"user_id"`
}

// UserTimelineEntriesArgs holds arguments for filtering user timeline entries.
type UserTimelineEntriesArgs struct {
	Status TimelineEntryStatus // To optionally filter by status
	FeedID string              // To optionally filter by feed
	Limit  uint64              // To optionally limit the number of entries returned
}

// TimelineEntryStatus represents the status of a timeline entry.
type TimelineEntryStatus string

const (
	TimelineEntryStatusRequiresJudgement TimelineEntryStatus = "requires_judgement"
	TimelineEntryStatusApproved          TimelineEntryStatus = "approved"
	TimelineEntryStatusRejected          TimelineEntryStatus = "rejected"
)
