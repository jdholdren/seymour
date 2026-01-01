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
func NewWorker(lc fx.Lifecycle, repo seymour.Repository, cli client.Client) (worker.Worker, error) {
	a := activities{
		repo: repo,
	}

	w := worker.New(cli, TaskQueue, worker.Options{})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := registerEverything(ctx, w, a, cli); err != nil {
				return fmt.Errorf("error registering workflows and activities: %T, %v", err, err)
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
	w.RegisterWorkflow(wfs.SyncAllFeeds)
	w.RegisterWorkflow(wfs.CreateFeed)
	w.RegisterWorkflow(wfs.RefreshUserTimeline)
	w.RegisterWorkflow(wfs.JudgeUserTimeline)
	w.RegisterWorkflow(wfs.RefreshAllUserTimelines)

	// Activities
	w.RegisterActivity(&a)

	// Schedules:
	// Sync RSS feeds
	handle := cli.ScheduleClient().GetHandle(ctx, "sync_all")
	if _, err := handle.Describe(ctx); err != nil {
		handle, err = cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: "sync_all",
			Spec: client.ScheduleSpec{
				Intervals: []client.ScheduleIntervalSpec{{Every: 15 * time.Minute}},
			},
			Action: &client.ScheduleWorkflowAction{
				ID:        "sync_all",
				Workflow:  wfs.SyncAllFeeds,
				TaskQueue: TaskQueue,
			},
			TriggerImmediately: true,
		})
		if err != nil {
			return err
		}
	}
	handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
	// Refresh timelines
	handle = cli.ScheduleClient().GetHandle(ctx, "refresh_timelines")
	if _, err := handle.Describe(ctx); err != nil {
		handle, err = cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: "refresh_timelines",
			Spec: client.ScheduleSpec{
				Intervals: []client.ScheduleIntervalSpec{{Every: 15 * time.Minute}},
			},
			Action: &client.ScheduleWorkflowAction{
				ID:        "refresh_timelines",
				Workflow:  wfs.RefreshAllUserTimelines, // TODO: Refresh all user timelines
				TaskQueue: TaskQueue,
			},
		})
		if err != nil {
			return err
		}
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
