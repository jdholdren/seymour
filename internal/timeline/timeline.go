package timeline

import "go.uber.org/fx"

type (
	// Subscription is an instance of a user's intention to listen to a feed.
	Subscription struct {
		ID     string `db:"id"`
		UserID string `db:"user_id"`
		FeedID string `db:"feed_id"`
	}
)

type (
	Service struct {
		repo Repo
	}

	Params struct {
		fx.In

		Repo Repo
	}
)

func NewService(p Params) Service {
	return Service{
		repo: p.Repo,
	}
}
