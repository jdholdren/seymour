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
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/agg"
	"github.com/jdholdren/seymour/internal/citadel"
	"github.com/jdholdren/seymour/internal/tempest"
	"github.com/jdholdren/seymour/internal/timeline"
	"github.com/jdholdren/seymour/logger"
)

type config struct {
	AggPort  int `env:"PORT, default=4445"`
	TimePort int `env:"PORT, default=4446"`

	Database string `env:"DATABASE, required"`

	// Which format to use for logging: either text or json
	LoggerFormat string `env:"LOGGER_FORMAT, default=text"`

	// Citadel stuffs
	CitadelPort        int    `env:"PORT, default=4444"`
	HTTPSCookies       bool   `env:"HTTPS_COOKIES, default=false"`
	GithubClientID     string `env:"GITHUB_CLIENT_ID, required"`
	GithubClientSecret string `env:"GITHUB_CLIENT_SECRET, required"`
	CookieHashKey      string `env:"COOKIE_HASH_KEY, required"`
	CookieBlockKey     string `env:"COOKIE_BLOCK_KEY, required"`
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
			citadel.Config{
				Port:               cfg.CitadelPort,
				CookieHashKey:      []byte(cfg.CookieHashKey),
				CookieBlockKey:     []byte(cfg.CookieBlockKey),
				GithubClientID:     cfg.GithubClientID,
				GithubClientSecret: cfg.GithubClientSecret,
				HttpsCookies:       cfg.HTTPSCookies,
			},
			cfg,
			fx.Annotate(ctx, fx.As(new(context.Context))),
		),
		fx.Provide(newDB),
		agg.Module,
		citadel.Module,
		tempest.Module,
		timeline.Module,
		fx.Invoke(func(citadel.Server) {}), // Start the BFF server
		fx.Invoke(func(worker.Worker) {}),  // Start the temporal worker
	).Run()
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
