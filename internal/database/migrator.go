package database

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

// RunMigrations performs all migrations in the given filesystem.
func RunMigrations(dbx *sqlx.DB, fs fs.FS, dirName string) error {
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
