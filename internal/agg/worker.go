package agg

import (
	"context"
	"time"

	"github.com/jdholdren/seymour/internal/agg/db"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const TaskQueue = "aggregator"

// RunWorker runs a Workflow and Activity worker for the Billing system.
func RunWorker(ctx context.Context, repo db.Repo, cli client.Client) error {
	a := activities{
		repo: repo,
	}

	w := worker.New(cli, TaskQueue, worker.Options{})

	// Workflows
	wfs := workflows{}
	w.RegisterWorkflow(wfs.SyncIndividual)
	w.RegisterWorkflow(wfs.SyncAll)
	w.RegisterWorkflow(wfs.CreateFeed)

	// Activities
	w.RegisterActivity(a.SyncFeed)
	w.RegisterActivity(a.AllFeeds)
	w.RegisterActivity(a.RemoveFeed)
	w.RegisterActivity(a.CreateFeed)

	// Schedules
	cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
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

	intChan := make(chan any)
	go func() {
		<-ctx.Done()
		close(intChan)
	}()

	return w.Run(intChan)
}
