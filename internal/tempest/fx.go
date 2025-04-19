package tempest

import "go.uber.org/fx"

var Module = fx.Module("tempest",
	fx.Provide(
		NewWorker,
	),
)
