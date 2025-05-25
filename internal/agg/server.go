// Package agg provides the aggregation daemon that scrapes feeds that
// it has been configured to.
package agg

import (
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/agg/db"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		repo db.Repo
	}

	Params struct {
		fx.In

		Repo db.Repo
	}
)

func NewServer(p Params) Server {
	return Server{
		repo: p.Repo,
	}
}
