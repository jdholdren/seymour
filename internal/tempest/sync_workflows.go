package tempest

import (
	"time"

	"log/slog"

	"github.com/jdholdren/seymour/internal/agg"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

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

	slog.Info("starting sync feed workflow", "feed_id", feedID)

	a := activities{}
	return workflow.ExecuteActivity(ctx, a.SyncRSSFeed, feedID).Get(ctx, nil)
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
	if err := workflow.ExecuteActivity(ctx, a.AllRSSFeeds).Get(ctx, &feeds); err != nil {
		slog.Error("failed to sync all feeds", "error", err)
		return err
	}

	wg := workflow.NewWaitGroup(ctx)
	wg.Add(len(feeds))
	for _, feed := range feeds {
		workflow.Go(ctx, func(ctx workflow.Context) {
			defer wg.Done()

			if err := workflow.ExecuteActivity(ctx, a.SyncRSSFeed, feed.ID).Get(ctx, nil); err != nil {
				slog.Error("failed to sync feed", "feed_id", feed.ID, "error", err)
			}
		})
	}

	wg.Wait(ctx)

	return nil
}
