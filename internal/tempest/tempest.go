package tempest

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/agg"
)

type (
	activities struct {
		agg agg.Service
	}

	workflows struct{}

	Params struct {
		fx.In

		Agg agg.Service
	}
)

func NewWorker(lc fx.Lifecycle, ctx context.Context, p Params) (worker.Worker, error) {
	a := activities{
		agg: p.Agg,
	}
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("error creating Temporal client: %s", err)
	}

	tw := worker.New(c, "tempest", worker.Options{})

	// Workflows
	w := workflows{}
	tw.RegisterWorkflow(w.SyncIndividual)
	tw.RegisterWorkflow(w.SyncAll)

	// Activities
	tw.RegisterActivity(a.SyncRSSFeed)
	tw.RegisterActivity(a.AllRSSFeeds)

	// Schedules
	c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: "sync_all",
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: 15 * time.Minute}},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "sync_all",
			Workflow:  w.SyncAll,
			TaskQueue: "tempest",
		},
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go tw.Run(worker.InterruptCh())
			return nil
		},
		OnStop: func(ctx context.Context) error {
			tw.Stop()
			c.Close()

			return nil
		},
	})

	return tw, nil
}
