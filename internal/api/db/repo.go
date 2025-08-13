package db

import "github.com/jmoiron/sqlx"

func NewRepo(db *sqlx.DB) Repo {
	return Repo{db: db}
}

type Repo struct {
	db *sqlx.DB
}
