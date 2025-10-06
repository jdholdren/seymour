package worker

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/seymour"
)

const TaskQueue = "shared"

// NewWorker sets up the worker with registration of workflows, activities, and schedules.
func NewWorker(lc fx.Lifecycle, feedService seymour.FeedService, tlService seymour.TimelineService, cli client.Client) (worker.Worker, error) {
	a := activities{
		feedService:     feedService,
		timelineService: tlService,
	}

	w := worker.New(cli, TaskQueue, worker.Options{})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := registerEverything(ctx, w, a, cli); err != nil {
				return fmt.Errorf("error registering workflows and activities: %s", err)
			}

			// Run the worker in a long-lived goroutine
			return w.Start()
		},
		OnStop: func(context.Context) error {
			w.Stop()
			return nil
		},
	})

	return w, nil
}

func registerEverything(ctx context.Context, w worker.Worker, a activities, cli client.Client) error {
	// Workflows
	wfs := workflows{}
	w.RegisterWorkflow(wfs.SyncIndividual)
	w.RegisterWorkflow(wfs.SyncAll)
	w.RegisterWorkflow(wfs.CreateFeed)
	w.RegisterWorkflow(wfs.RefreshTimelines)
	w.RegisterWorkflow(wfs.JudgeUserTimeline)

	// Activities
	//
	// TODO(jdh): Some of these are too granular, make them more action-based and not have so much
	// schema in there.
	w.RegisterActivity(a.SyncFeed)
	w.RegisterActivity(a.AllFeeds)
	w.RegisterActivity(a.RemoveFeed)
	w.RegisterActivity(a.CreateFeed)
	w.RegisterActivity(a.Feed)
	w.RegisterActivity(a.InsertMissingTimelineEntries)
	w.RegisterActivity(a.JudgeEntries)
	w.RegisterActivity(a.MarkEntriesAsJudged)

	// Schedules:
	// Sync RSS feeds
	handle, err := cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: "sync_all",
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: 15 * time.Minute}},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "sync_all",
			Workflow:  wfs.SyncAll,
			TaskQueue: TaskQueue,
		},
	})
	if err != nil {
		return err
	}
	handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
	// Refresh timelines
	handle, err = cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: "refresh_timelines",
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: 15 * time.Minute}},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "refresh_timelines",
			Workflow:  wfs.RefreshTimelines,
			TaskQueue: TaskQueue,
		},
	})
	if err != nil {
		return err
	}
	handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})

	return nil
}
