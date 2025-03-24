package v1

import (
	"github.com/jdholdren/seymour/api"
)

type CreateFeedRequest struct {
	URL string `json:"url"`
}

type CreateFeedResponse struct {
	ID string `json:"id"`
}

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
