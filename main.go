// Seymour-Agg is the rss feed aggregator.
//
// It fetches feed entries from the various rss feeds that it has been
// told about.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/sync/errgroup"

	"github.com/jdholdren/seymour/internal/agg"
	aggdb "github.com/jdholdren/seymour/internal/agg/database"
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
	if err := run(ctx, cfg); err != nil {
		slog.Error("error running", "error", err)
		os.Exit(1)
	}
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func run(ctx context.Context, cfg config) error {
	slog.Info("running", "config", cfg)

	// Connect to the db
	dbx, err := sqlx.Open("sqlite", fmt.Sprintf("%s?_jounral_mode=WAL", cfg.Database))
	if err != nil {
		return fmt.Errorf("error opening database: %s", err)
	}

	// Migrate, always
	if err := migrateDB(dbx); err != nil {
		return fmt.Errorf("error migrating: %s", err)
	}

	aggRepo := aggdb.NewRepo(dbx)
	s := agg.NewServer(cfg.Port, aggRepo)
	syncer := agg.NewSyncer(aggRepo)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// Start the server
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("error listening: %s", err)
		}

		return nil
	})
	g.Go(func() error {
		// Block from shutting down until the group is canceled
		<-gCtx.Done()

		downCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := s.Shutdown(downCtx); err != nil {
			slog.Error("error shutting down server", "error", err)
		}

		return nil
	})

	g.Go(func() error {
		// Start the syncer
		if err := syncer.Run(gCtx); err != nil {
			return fmt.Errorf("error running syncer: %s", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("error running: %s", err)
	}

	return nil
}

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
