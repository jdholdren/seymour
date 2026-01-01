package sqlite

import (
	"github.com/jmoiron/sqlx"

	"github.com/jdholdren/seymour/internal/seymour"
)

// Ensure Repo implements the Repository interface
var _ seymour.Repository = (*Repo)(nil)

type Repo struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) Repo {
	return Repo{db: db}
}
