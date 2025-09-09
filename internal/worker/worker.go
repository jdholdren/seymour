package worker

import (
	"context"
	"time"

	"github.com/jdholdren/seymour/internal/seymour"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const TaskQueue = "shared"

// RunWorker runs a Workflow and Activity worker for the Billing system.
func RunWorker(ctx context.Context, feedService seymour.FeedService, tlService seymour.TimelineService, cli client.Client) error {
	a := activities{
		feedService:     feedService,
		timelineService: tlService,
	}

	w := worker.New(cli, TaskQueue, worker.Options{})

	// Workflows
	wfs := workflows{}
	w.RegisterWorkflow(wfs.SyncIndividual)
	w.RegisterWorkflow(wfs.SyncAll)
	w.RegisterWorkflow(wfs.CreateFeed)
	w.RegisterWorkflow(wfs.RefreshTimelines)
	w.RegisterWorkflow(wfs.JudgeUserTimeline)

	// Activities
	w.RegisterActivity(a.SyncFeed)
	w.RegisterActivity(a.AllFeeds)
	w.RegisterActivity(a.RemoveFeed)
	w.RegisterActivity(a.CreateFeed)
	w.RegisterActivity(a.Feed)
	w.RegisterActivity(a.InsertMissingTimelineEntries)
	w.RegisterActivity(a.NeedingJudgement)
	w.RegisterActivity(a.JudgeEntries)
	w.RegisterActivity(a.MarkEntriesAsJudged)

	// Schedules:
	// Sync RSS feeds
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
	// Refresh timelines
	cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
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

	intChan := make(chan any)
	go func() {
		<-ctx.Done()
		close(intChan)
	}()

	return w.Run(intChan)
}
