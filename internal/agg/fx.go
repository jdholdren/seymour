package agg

import (
	"go.uber.org/fx"
)

var Module = fx.Module("agg",
	fx.Provide(
		NewAggregator,
		NewRepo,
	),
)
