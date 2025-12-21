package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sethvargo/go-envconfig"
	"github.com/sethvargo/go-retry"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
	_ "golang.org/x/crypto/x509roots/fallback"
	_ "modernc.org/sqlite"

	"github.com/jdholdren/seymour/internal/logger"
	"github.com/jdholdren/seymour/internal/seymour"
	seyqlite "github.com/jdholdren/seymour/internal/sqlite"
	seyworker "github.com/jdholdren/seymour/internal/worker"
)

type config struct {
	Database         string `env:"DATABASE, required"`
	TemporalHostPort string `env:"TEMPORAL_HOST_PORT, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Parse the config
	var cfg config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatalf("error parsing config: %s", err)
	}

	l := slog.New(logger.NewContextHandler(slog.NewTextHandler(os.Stdout, nil)))
	slog.SetDefault(l)

	// Connect to the sqlite db
	dbx, err := sqlx.Open("sqlite", cfg.Database)
	if err != nil {
		log.Fatalf("error opening database: %s", err)
	}
	defer dbx.Close()

	repo := seyqlite.New(dbx)

	// Retry until temporal is ready
	var temporalCli client.Client
	if err := retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		c, err := client.Dial(client.Options{
			HostPort: cfg.TemporalHostPort,
			Logger:   slog.Default(),
		})
		if err != nil {
			return retry.RetryableError(err)
		}
		temporalCli = c

		return nil
	}); err != nil {
		log.Fatalln("Unable to create Temporal client:", err)
	}

	fx.New(
		fx.Supply(
			dbx,
			fx.Annotate(ctx, fx.As(new(context.Context))),
			fx.Annotate(temporalCli, fx.As(new(client.Client))),
			fx.Annotate(repo, fx.As(new(seymour.FeedService))),
			fx.Annotate(repo, fx.As(new(seymour.TimelineService))),
			fx.Annotate(repo, fx.As(new(seymour.UserService))),
		),
		fx.Provide(seyworker.NewWorker),
		fx.Invoke(func(
			ctx context.Context,
			a seymour.FeedService,
			c client.Client,
			t seymour.TimelineService,
			w worker.Worker,
		) {
			// Start the worker
		}),
	).Run()
}
