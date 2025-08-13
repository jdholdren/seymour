// Package api provides the BFF server for the client side application.
//
// It's the main, monolithic package that handles most of all of the wiring for the whole app.
// It will use the other packages to provide users with functionality.
package api

import (
	"go.uber.org/fx"
)

var Module = fx.Module("api",
	fx.Provide(
		NewServer,
	),
)
