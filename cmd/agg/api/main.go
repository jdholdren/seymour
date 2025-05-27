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

	"github.com/jdholdren/seymour/internal/agg"
	"github.com/jdholdren/seymour/internal/agg/migrations"
	"github.com/jdholdren/seymour/internal/logger"
	"github.com/jdholdren/seymour/internal/migrator"
)

type config struct {
	Database         string `env:"DATABASE, required"`
	Port             int    `env:"PORT, default=4444"`
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
	//
	// Run all migrations
	if err := migrator.RunMigrations(dbx, migrations.Migrations, "."); err != nil {
		log.Fatalf("error running migrations: %s", err)
	}

	// Dial temporal
	c, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHostPort,
	})
	if err != nil {
		log.Fatalln("error creating Temporal client:", err)
	}

	// Start the application
	fx.New(
		fx.Supply(
			agg.ServerConfig{
				Port: cfg.Port,
			},
			dbx,
			fx.Annotate(ctx, fx.As(new(context.Context))),
			fx.Annotate(c, fx.As(new(client.Client))),
		),
		agg.Module,
		fx.Invoke(func(agg.Server) {}), // Start the agg server
	).Run()
}
