package v1

import (
	"net/http"
	"time"

	"github.com/jdholdren/seymour/errors"
)

type (
	CreateFeedRequest struct {
		URL string `json:"url"`
	}

	CreateFeedResponse struct {
		ID string `json:"id"`
	}

	Feed struct {
		ID           string     `json:"id"`
		Title        string     `json:"title"`
		Description  string     `json:"description"`
		LastSyncedAt *time.Time `json:"last_synced_at"`
		CreatedAt    time.Time  `json:"created_at"`
		UpdatedAt    time.Time  `json:"updated_at"`
	}

	Entry struct {
		ID          string `json:"id"`
		FeedID      string `json:"feed_id"`
		GUID        string `json:"guid"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
)

// Validate checks that the body (minus logic checks) is valid.
//
// Returns an api.Error if the request is invalid.
func (r CreateFeedRequest) Validate() error {
	errs := []errors.Detail{}
	if r.URL == "" {
		errs = append(errs, errors.Detail{
			Field: "url",
			Error: "url is required",
		})
	}
	if len(errs) > 0 {
		return errors.E("request was invalid", http.StatusBadRequest, errs)
	}

	return nil
}
