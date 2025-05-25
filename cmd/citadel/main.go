package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/jmoiron/sqlx"
	"github.com/sethvargo/go-envconfig"
	"go.uber.org/fx"
	_ "modernc.org/sqlite"

	"github.com/jdholdren/seymour/internal/citadel"
	"github.com/jdholdren/seymour/internal/citadel/migrations"
	"github.com/jdholdren/seymour/internal/database"
	"github.com/jdholdren/seymour/internal/logger"
)

type config struct {
	Database string `env:"DATABASE, required"`

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
	dbx, err := sqlx.Open("sqlite", cfg.Database)
	if err != nil {
		log.Fatalf("error opening database: %s", err)
	}
	defer dbx.Close()

	// Run all migrations
	if err := database.RunMigrations(dbx, migrations.Migrations, "."); err != nil {
		log.Fatalf("error running migrations: %s", err)
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
		),
		citadel.Module,
		fx.Invoke(func(citadel.Server) {}), // Start the BFF server
	).Run()

}
