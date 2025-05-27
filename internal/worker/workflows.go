package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/jdholdren/seymour/internal/agg"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
)

type workflows struct{}

func (workflows) SyncIndividual(ctx workflow.Context, feedID string) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3, // 0 is unlimited retries
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	return workflow.ExecuteActivity(ctx, acts.SyncFeed, feedID).Get(ctx, nil)
}

func (workflows) SyncAll(ctx workflow.Context) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	a := activities{}

	var feeds []agg.Feed
	if err := workflow.ExecuteActivity(ctx, acts.AllFeeds).Get(ctx, &feeds); err != nil {
		slog.Error("failed to sync all feeds", "error", err)
		return err
	}

	wg := workflow.NewWaitGroup(ctx)
	wg.Add(len(feeds))
	for _, feed := range feeds {
		workflow.Go(ctx, func(ctx workflow.Context) {
			defer wg.Done()

			if err := workflow.ExecuteActivity(ctx, a.SyncFeed, feed.ID).Get(ctx, nil); err != nil {
				slog.Error("failed to sync feed", "feed_id", feed.ID, "error", err)
			}
		})
	}

	wg.Wait(ctx)

	return nil
}

func TriggerCreateFeedWorkflow(ctx context.Context, c client.Client, feedURL string) (string, error) {
	options := client.StartWorkflowOptions{
		TaskQueue: TaskQueue,
	}
	we, err := c.ExecuteWorkflow(ctx, options, workflows{}.CreateFeed, feedURL)
	if err != nil {
		return "", fmt.Errorf("unable to execute workflow: %s", err)
	}

	var feedID string
	err = we.Get(context.Background(), &feedID)
	seyErr := &seyerrs.Error{}
	if asSeyerr(err, &seyErr) {
		return "", seyErr
	}
	if err != nil {
		return "", fmt.Errorf("error executing workflow: %s", err)
	}

	return feedID, nil
}

// CreateFeed inserts a new feed, tries to sync, and rolls back if it's unable to.
//
// Returns the ID of the created feed.
func (workflows) CreateFeed(ctx workflow.Context, feedURL string) (string, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Insert the feed
	var feedID string
	if err := workflow.ExecuteActivity(ctx, acts.CreateFeed, feedURL).Get(ctx, &feedID); err != nil {
		slog.Error("failed to create feed", "error", err)
		return "", err
	}

	// Make sure the feed hasn't already been synced
	var feed agg.Feed
	if err := workflow.ExecuteActivity(ctx, acts.Feed, feedID).Get(ctx, &feed); err != nil {
		slog.Error("failed to fetch feed", "error", err)
		return "", err
	}
	if feed.LastSyncedAt != nil { // Exit early
		slog.Info("feed already synced", "feed_id", feedID)
		return feedID, nil
	}

	// Sync the feed
	err := workflow.ExecuteActivity(ctx, acts.SyncFeed, feedID).Get(ctx, nil)
	if err != nil {
		slog.Error("failed to sync feed", "feed_id", feedID, "error", err)
		// If there's an issue syncing, remove the feed

		if err := workflow.ExecuteActivity(ctx, acts.RemoveFeed, feedID).Get(ctx, nil); err != nil {
			slog.Error("failed to remove feed", "feed_id", feedID, "error", err)
			return "", err
		}

		return "", err
	}

	return feedID, nil
}
