// Seymour-Agg is the rss feed aggregator.
//
// It fetches feed entries from the various rss feeds that it has been
// told about.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sethvargo/go-envconfig"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/agg"
	"github.com/jdholdren/seymour/logger"
)

type config struct {
	Port     int    `env:"PORT, default=4444"`
	Database string `env:"DATABASE, required"`

	// Which format to use for logging: either text or json
	LoggerFormat string `env:"LOGGER_FORMAT, default=text"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Parse the config
	var cfg config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatalf("error parsing config: %s", err)
	}

	// Determine which logger format to use
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, nil)
	if cfg.LoggerFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, nil)
	}
	l := slog.New(logger.NewContextHandler(handler))
	slog.SetDefault(l)

	// Start the application
	fx.New(
		fx.Supply(
			agg.Config{
				Port: cfg.Port,
			},
			cfg,
			fx.Annotate(ctx, fx.As(new(context.Context))),
		),
		fx.Provide(newDB),
		agg.Module,
		fx.Invoke(func(agg.Server) {}), // Always start the agg server
	).Run()

	panic("")
}

func newDB(lc fx.Lifecycle, cfg config) (*sqlx.DB, error) {
	dbx, err := sqlx.Open("sqlite", fmt.Sprintf("%s?_jounral_mode=WAL", cfg.Database))
	if err != nil {
		return nil, fmt.Errorf("error opening database: %s", err)
	}

	lc.Append(fx.Hook{
		// Run migrations on startup:
		OnStart: func(context.Context) error {
			return migrateDB(dbx)
		},
		// Close the DB connection on shutdown:
		OnStop: func(context.Context) error {
			return dbx.Close()
		},
	})

	return dbx, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func migrateDB(dbx *sqlx.DB) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("error creating migrations source: %s", err)
	}
	i, err := sqlite.WithInstance(dbx.DB, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("error creating sqlite instance for migration: %s", err)
	}
	migrator, err := migrate.NewWithInstance("iofs", d, "sqlite3", i)
	if err != nil {
		return fmt.Errorf("error creating migrator: %s", err)
	}
	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error migrating: %s", err)
	}
	slog.Info("migrated")

	return nil
}
