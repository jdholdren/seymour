package timeline

import "go.uber.org/fx"

var Module = fx.Module(
	"timeline",
	fx.Provide(NewServer),
)
