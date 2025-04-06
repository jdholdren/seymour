package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error represents a universal error type between the services.
type Error struct {
	Status  int      `json:"-"`
	Err     error    `json:"message"` // The error this wraps
	Details []Detail `json:"details"`
}

type Detail struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s, details: %v", e.Status, e.Err, e.Details)
}

func E(args ...any) *Error {
	ret := &Error{
		Status:  http.StatusInternalServerError,
		Err:     nil,
		Details: nil,
	}

	for _, arg := range args {
		switch arg := arg.(type) {
		case string:
			ret.Err = errors.New(arg)
		case error:
			ret.Err = arg
		case int:
			ret.Status = arg
		case Detail:
			ret.Details = append(ret.Details, arg)
		case []Detail:
			ret.Details = append(ret.Details, arg...)
		}
	}

	return ret
}
