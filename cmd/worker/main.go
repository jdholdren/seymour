package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/jmoiron/sqlx"
	"github.com/sethvargo/go-envconfig"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	_ "golang.org/x/crypto/x509roots/fallback"
	_ "modernc.org/sqlite"

	"github.com/jdholdren/seymour/internal/agg"
	"github.com/jdholdren/seymour/internal/logger"
	"github.com/jdholdren/seymour/internal/worker"
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

	c, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client:", err)
	}

	fx.New(
		fx.Supply(
			dbx,
			fx.Annotate(ctx, fx.As(new(context.Context))),
			fx.Annotate(c, fx.As(new(client.Client))),
		),
		agg.Module,
		fx.Invoke(func(ctx context.Context, a agg.Aggregator, c client.Client) {
			worker.RunWorker(ctx, a, c)
		}), // Start the worker
	).Run()
}
