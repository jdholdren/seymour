package v1

import (
	"net/http"
	"time"

	seyerrs "github.com/jdholdren/seymour/errors"
)

type (
	CreateSubscriptionRequest struct {
		FeedID string `json:"feed_id"`
	}

	Subscription struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
		FeedID    string    `json:"feed_id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
)

func (c CreateSubscriptionRequest) Validate() error {
	var errs []seyerrs.Detail
	if c.FeedID == "" {
		errs = append(errs, seyerrs.Detail{Field: "feed_id", Error: "required"})
	}
	if len(errs) > 0 {
		return seyerrs.E("invalid request", http.StatusBadRequest, errs)
	}

	return nil
}
