// Package timeline provides the API and workers for the timeline service.
//
// This service is responsible for managing a user's timeline and subscriptions.
// Any configuration of the timeline is stored here.
//
// This service heavily uses the aggregator to read posts to batch submit to the
// evaluator.
package timeline

import "go.uber.org/fx"

var Module = fx.Module("timeline",
	fx.Provide(
		NewService,
		NewRepo,
	),
)
