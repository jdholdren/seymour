package citadel

import (
	"go.uber.org/fx"
)

var Module = fx.Module("citadel",
	fx.Provide(
		NewServer,
		NewUserRepo,
	),
)
