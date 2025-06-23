package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/sethvargo/go-envconfig"
	"github.com/sethvargo/go-retry"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	_ "modernc.org/sqlite"

	"github.com/jdholdren/seymour/internal/citadel"
	"github.com/jdholdren/seymour/internal/logger"
	"github.com/jdholdren/seymour/internal/migrations"
	"github.com/jdholdren/seymour/internal/seymour"
	seyqlite "github.com/jdholdren/seymour/internal/sqlite"
)

type config struct {
	Database         string `env:"DATABASE, required"`
	TemporalHostPort string `env:"TEMPORAL_HOST_PORT, required"`

	Port               int    `env:"PORT, default=4444"`
	HTTPSCookies       bool   `env:"HTTPS_COOKIES, default=false"`
	GithubClientID     string `env:"GITHUB_CLIENT_ID"`
	GithubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	CookieHashKey      string `env:"COOKIE_HASH_KEY"`
	CookieBlockKey     string `env:"COOKIE_BLOCK_KEY"`
	DebugEndpoints     bool   `env:"DEBUG_ENDPOINTS, default=false"`
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
	dbx, err := sqlx.Open("sqlite", fmt.Sprintf("%s?_txlock=immediate&_journal_mode=WAL&_busy_timeout=5000", cfg.Database))
	if err != nil {
		log.Fatalf("error opening database: %s", err)
	}
	defer dbx.Close()

	// Run all migrations
	if err := runMigrations(dbx, migrations.Migrations, "."); err != nil {
		log.Fatalf("error running migrations: %s", err)
	}

	repo := seyqlite.New(dbx)

	// Retry until temporal is ready
	var temporalCli client.Client
	if err := retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		c, err := client.Dial(client.Options{
			HostPort: cfg.TemporalHostPort,
		})
		if err != nil {
			return retry.RetryableError(err)
		}
		temporalCli = c

		return nil
	}); err != nil {
		log.Fatalln("Unable to create Temporal client:", err)
	}

	// Start the application
	fx.New(
		fx.Supply(
			citadel.ServerConfig{
				Port:               cfg.Port,
				GithubClientID:     cfg.GithubClientID,
				GithubClientSecret: cfg.GithubClientSecret,
				CookieHashKey:      []byte(cfg.CookieHashKey),
				CookieBlockKey:     []byte(cfg.CookieBlockKey),
				HttpsCookies:       cfg.HTTPSCookies,
				DebugEndpoints:     cfg.DebugEndpoints,
			},
			dbx,
			fx.Annotate(ctx, fx.As(new(context.Context))),
			fx.Annotate(temporalCli, fx.As(new(client.Client))),
			fx.Annotate(repo, fx.As(new(seymour.FeedRepo))),
			fx.Annotate(repo, fx.As(new(seymour.TimelineRepo))),
		),
		citadel.Module,
		fx.Invoke(func(citadel.Server) {}), // Start the BFF server
	).Run()
}

// Performs all migrations in the given filesystem.
func runMigrations(dbx *sqlx.DB, fs fs.FS, dirName string) error {
	d, err := iofs.New(fs, dirName)
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
