// Package citadel provides the BFF server for the client side application.
//
// It's the main, monolithic package that handles most of all of the wiring for the whole app.
// It will use the other packages to provide users with functionality.
package citadel

import (
	"go.uber.org/fx"
)

var Module = fx.Module("citadel",
	fx.Provide(
		NewServer,
	),
)
