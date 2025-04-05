package v1

import (
	"time"

	"github.com/jdholdren/seymour/api"
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
	errs := []api.ErrorDetail{}
	if r.URL == "" {
		errs = append(errs, api.ErrorDetail{
			Field: "url",
			Error: "url is required",
		})
	}
	if len(errs) > 0 {
		return api.Error{
			Reason:  "invalid_request",
			Message: "request was invalid",
			Details: errs,
		}
	}

	return nil
}
