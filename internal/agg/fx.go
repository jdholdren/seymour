package agg

import (
	"github.com/jdholdren/seymour/internal/agg/database"
	"go.uber.org/fx"
)

var Module = fx.Module("aggregator",
	fx.Provide(
		NewServer,
		database.NewRepo,
		NewSyncer,
	),
)
