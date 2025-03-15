package api

import "fmt"

// Error represents a universal error type between the services.
type Error struct {
	Reason  string        `json:"reason"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

type ErrorDetail struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}
