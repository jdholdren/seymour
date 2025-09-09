package worker

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/jdholdren/seymour/internal/seymour"
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
	l := workflow.GetLogger(ctx)

	var feeds []seymour.Feed
	if err := workflow.ExecuteActivity(ctx, acts.AllFeeds).Get(ctx, &feeds); err != nil {
		l.Error("failed to sync all feeds", "error", err)
		return err
	}

	wg := workflow.NewWaitGroup(ctx)
	wg.Add(len(feeds))
	for _, feed := range feeds {
		workflow.Go(ctx, func(ctx workflow.Context) {
			defer wg.Done()

			if err := workflow.ExecuteActivity(ctx, acts.SyncFeed, feed.ID).Get(ctx, nil); err != nil {
				l.Error("failed to sync feed", "feed_id", feed.ID, "error", err)
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
	l := workflow.GetLogger(ctx)

	// Insert the feed
	var feedID string
	if err := workflow.ExecuteActivity(ctx, acts.CreateFeed, feedURL).Get(ctx, &feedID); err != nil {
		l.Error("failed to create feed", "error", err)
		return "", err
	}

	// Make sure the feed hasn't already been synced
	var feed seymour.Feed
	if err := workflow.ExecuteActivity(ctx, acts.Feed, feedID).Get(ctx, &feed); err != nil {
		l.Error("failed to fetch feed", "error", err)
		return "", err
	}
	if feed.LastSyncedAt != nil { // Exit early
		l.Info("feed already synced", "feed_id", feedID)
		return feedID, nil
	}

	// Sync the feed
	err := workflow.ExecuteActivity(ctx, acts.SyncFeed, feedID).Get(ctx, nil)
	if err != nil {
		l.Error("failed to sync feed", "feed_id", feedID, "error", err)

		// If there's an issue syncing, remove the feed
		if err := workflow.ExecuteActivity(ctx, acts.RemoveFeed, feedID).Get(ctx, nil); err != nil {
			l.Error("failed to remove feed", "feed_id", feedID, "error", err)
			return "", err
		}

		return "", err
	}

	return feedID, nil
}

// This workflow grabs all the different subscriptions for users and makes sure that they've
// had the possible entries put into their timeline.
//
// Should also kick off workflows to judge entries for each user.
func (workflows) RefreshTimelines(ctx workflow.Context) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	l := workflow.GetLogger(ctx)

	var users []string
	if err := workflow.ExecuteActivity(ctx, acts.InsertMissingTimelineEntries).Get(ctx, &users); err != nil {
		l.Error("failed to insert missing timeline entries", "error", err)
		return err
	}

	// Refresh the timelines for each user
	for _, userID := range users {
		// Start child workflow to judge each member's timeline
		ctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			// Ensure only one judgement at a time, allow current one to process
			WorkflowID:            "judge-user-timeline-" + userID,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING,
			ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
			TaskQueue:             TaskQueue,
		})
		if err := workflow.ExecuteChildWorkflow(ctx, workflows.JudgeUserTimeline, userID).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
			l.Error("failed to start child workflow", "error", err)
			return err
		}
	}

	return nil
}

func (workflows) JudgeUserTimeline(ctx workflow.Context, userID string) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	l := workflow.GetLogger(ctx)

	// Get all entries needing judgement
	var entries []seymour.TimelineEntryWithFeed
	if err := workflow.ExecuteActivity(ctx, acts.NeedingJudgement, userID).Get(ctx, &entries); err != nil {
		l.Error("failed to get entries needing judgement", "error", err)
		return err
	}

	// Judge entries
	var j judgements
	if err := workflow.ExecuteActivity(ctx, acts.JudgeEntries, userID, entries).Get(ctx, &j); err != nil {
		l.Error("failed to judge entries", "error", err)
		return err
	}

	// Save the judgements
	if err := workflow.ExecuteActivity(ctx, acts.MarkEntriesAsJudged, j).Get(ctx, nil); err != nil {
		l.Error("failed to save judgements", "error", err)
		return err
	}

	return nil
}
