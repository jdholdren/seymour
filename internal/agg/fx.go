package agg

import (
	"github.com/jdholdren/seymour/internal/agg/db"
	"go.uber.org/fx"
)

var Module = fx.Module("agg",
	fx.Provide(
		NewAggregator,
		db.NewRepo,
	),
)
