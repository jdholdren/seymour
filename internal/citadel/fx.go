// Package citadel provides the BFF server for the client side application.
//
// It also handles common functions like user signup and auth in its own state.
package citadel

import (
	"go.uber.org/fx"
)

var Module = fx.Module("citadel",
	fx.Provide(
		NewServer,
	),
)
