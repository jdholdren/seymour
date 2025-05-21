package timeline

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) Repo {
	return Repo{
		db: db,
	}
}

func InsertSubscription(ctx context.Context, sub Subscription) error {
	return nil
}
