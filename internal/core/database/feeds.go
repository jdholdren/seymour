package database

import "github.com/jmoiron/sqlx"

// Repo represents the surface for interacting with feeds.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates a new instance of Repo.
func NewRepo(dbx *sqlx.DB) Repo {
	return Repo{
		db: dbx,
	}
}
