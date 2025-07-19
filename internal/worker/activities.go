package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jdholdren/seymour/internal/seymour"
	"github.com/jdholdren/seymour/internal/sync"
	"go.temporal.io/sdk/activity"
)

type activities struct {
	feedService     seymour.FeedService
	timelineService seymour.TimelineService
}

// Instance to make the workflow a bit more readable
var acts = activities{}

// Fetches all RSS feeds we know about in the system.
func (a activities) AllFeeds(ctx context.Context) ([]seymour.Feed, error) {
	feeds, err := a.feedService.AllFeeds(ctx)
	if err != nil {
		return nil, err
	}

	return feeds, nil
}

// Fetches a single feed
func (a activities) Feed(ctx context.Context, feedID string) (seymour.Feed, error) {
	feed, err := a.feedService.Feed(ctx, feedID)
	if err != nil {
		return seymour.Feed{}, err
	}

	return feed, nil
}

// Goes to the url and grabs the RSS feed items.
func (a activities) SyncFeed(ctx context.Context, feedID string) error {
	feed, err := a.feedService.Feed(ctx, feedID)
	if err != nil {
		return err
	}

	feed, entries, err := sync.Feed(ctx, feed.ID, feed.URL)
	if err != nil {
		return err
	}

	if err := a.feedService.UpdateFeed(ctx, feed.ID, seymour.UpdateFeedArgs{
		Title:       *feed.Title,
		Description: *feed.Title,
		LastSynced:  time.Now(),
	}); err != nil {
		return err
	}
	if err := a.feedService.InsertEntries(ctx, entries); err != nil {
		return err
	}

	return err
}

func (a activities) CreateFeed(ctx context.Context, feedURL string) (string, error) {
	feed, err := a.feedService.InsertFeed(ctx, feedURL)
	if errors.Is(err, seymour.ErrConflict) {
		// Fetch the feed from the database
		feed, err = a.feedService.FeedByURL(ctx, feedURL)
		if err != nil {
			return "", fmt.Errorf("error fetching conflicting feed: %s", err)
		}

		return feed.ID, nil
	}
	if err != nil {
		return "", fmt.Errorf("error inserting feed: %w", err)
	}

	slog.Debug("inserted feed", "feedID", feed.ID)

	return feed.ID, nil
}

func (a activities) RemoveFeed(ctx context.Context, feedID string) error {
	if err := a.feedService.DeleteFeed(ctx, feedID); err != nil {
		return fmt.Errorf("error deleting feed: %w", err)
	}

	return nil
}

// Inserts timeline entries that should be present in a user's timeline based on subscription but are missing.
//
// Returns a list of the affected users.
func (a activities) InsertMissingTimelineEntries(ctx context.Context) ([]string, error) {
	l := activity.GetLogger(ctx)

	missing, err := a.timelineService.MissingEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding missing timeline entries: %s", err)
	}

	l.Info("searched for missing timeline entries", "length", len(missing))

	// Keep track of affected users
	userIDs := make(map[string]struct{})
	for _, m := range missing {
		if err := a.timelineService.InsertEntry(ctx, seymour.TimelineEntry{
			UserID:      m.UserID,
			FeedEntryID: m.FeedEntryID,
			Status:      seymour.TimelineEntryStatusRequiresJudgement,
		}); err != nil {
			return nil, fmt.Errorf("error inserting timeline entry: %w", err)
		}

		userIDs[m.UserID] = struct{}{}
	}

	// Turn the map into a slice of strings
	users := make([]string, 0, len(userIDs))
	for userID := range userIDs {
		users = append(users, userID)
	}

	return users, nil
}

func (a activities) NeedingJudgement(ctx context.Context, userID string) ([]seymour.TimelineEntryWithFeed, error) {
	entries, err := a.timelineService.EntriesNeedingJudgement(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error finding needing judgement timeline entries: %s", err)
	}

	return entries, nil
}

// Type that holds a timeline entry ID and whether it has been approved.
type judgements map[string]bool

func (a activities) JudgeEntries(ctx context.Context, userID string, entries []seymour.TimelineEntryWithFeed) (judgements, error) {
	// TODO: Send to AI for judgement, if user has an AI configuration

	// For now, just approve all entries
	j := make(judgements)
	for _, entry := range entries {
		j[entry.TimelineEntryID] = true
	}

	return j, nil
}

func (a activities) MarkEntriesAsJudged(ctx context.Context, js judgements) error {
	for timelineEntryID, approved := range js {
		status := seymour.TimelineEntryStatusRejected
		if approved {
			status = seymour.TimelineEntryStatusApproved
		}

		if err := a.timelineService.UpdateTimelineEntry(ctx, timelineEntryID, status); err != nil {
			return fmt.Errorf("error updating timeline entry status: %w", err)
		}
	}

	return nil
}
