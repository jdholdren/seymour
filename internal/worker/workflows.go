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
)

type workflows struct{}

func (workflows) SyncAllFeeds(ctx workflow.Context) error {
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

	var allFeedCount int
	if err := workflow.ExecuteActivity(ctx, acts.CountAllFeeds).Get(ctx, &allFeedCount); err != nil {
		l.Error("failed to sync all feeds", "error", err)
		return err
	}

	// Batch each one into a group to sync
	batches := allFeedCount / 50
	if batches*50 < allFeedCount {
		batches += 1
	}

	wg := workflow.NewWaitGroup(ctx)
	for i := range batches {
		// Get a page of feed ID's
		var ids []string
		if err := workflow.ExecuteActivity(ctx, acts.FeedIDPage, i*50, 50).Get(ctx, &ids); err != nil {
			l.Error("failed to get feed IDs", "error", err)
			return err
		}
		wg.Add(len(ids))

		for _, id := range ids {
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := workflow.ExecuteActivity(ctx, acts.SyncFeed, id, true).Get(ctx, nil); err != nil {
					l.Error("failed to sync feed", "feed_id", id, "error", err)
				}
			})
		}

	}
	wg.Wait(ctx)

	return nil
}

func TriggerCreateFeedWorkflow(ctx context.Context, c client.Client, feedURL, userID string) (string, error) {
	options := client.StartWorkflowOptions{
		TaskQueue: TaskQueue,
	}
	we, err := c.ExecuteWorkflow(ctx, options, workflows{}.CreateFeed, feedURL, userID)
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
func (workflows) CreateFeed(ctx workflow.Context, feedURL, userID string) (string, error) {
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

	// Trigger a refresh of that user's timeline
	ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		// Ensure only one judgement at a time, allow current one to process
		WorkflowID:            "refresh-user-timeline-" + userID,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING,
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		TaskQueue:             TaskQueue,
	})
	if err := workflow.ExecuteChildWorkflow(ctx, workflows.RefreshUserTimeline, userID).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		l.Error("failed to start child workflow", "error", err)
		return "", err
	}

	return feedID, nil
}

func (workflows) RefreshAllUserTimelines(ctx workflow.Context) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	l := workflow.GetLogger(ctx)

	// Get all user ids
	var userIDs []string
	if err := workflow.ExecuteActivity(ctx, acts.AllUserIDs).Get(ctx, &userIDs); err != nil {
		return err
	}

	l.Info("starting user timeline refresh", "total_users", len(userIDs))

	// Process users with controlled concurrency to avoid overwhelming Temporal
	wg := workflow.NewWaitGroup(ctx)
	wg.Add(len(userIDs))
	for _, id := range userIDs {
		workflow.Go(ctx, func(ctx workflow.Context) {
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
			})
			if err := workflow.ExecuteChildWorkflow(childCtx, workflows.RefreshUserTimeline, id).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
				l.Error("failed to refresh user timeline", "error", err)
			}
			wg.Done()
		})
	}

	wg.Wait(ctx)
	l.Info("completed user timeline refresh", "total_users", len(userIDs))
	return nil
}

// RefreshUserTimeline syncs any missing entries for the user based on
// their subscriptions, and then judges their new timeline.
func (workflows) RefreshUserTimeline(ctx workflow.Context, userID string) error {
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

	var missingEntryCount int
	if err := workflow.ExecuteActivity(ctx, acts.InsertMissingTimelineEntries, userID).Get(ctx, &missingEntryCount); err != nil {
		l.Error("failed to insert missing timeline entries", "error", err)
		return err
	}

	// If no entries added, just exit early
	l.Debug("no entries added, return early")
	if missingEntryCount == 0 {
		return nil
	}

	// Start child workflow to judge the user's timeline
	ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
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

	// Judge entries
	var j judgements
	if err := workflow.ExecuteActivity(ctx, acts.JudgeEntries, userID).Get(ctx, &j); err != nil {
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
