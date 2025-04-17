package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Error represents a universal error type between the services.
type Error struct {
	Status  int
	Err     error // The error this wraps
	Details []Detail
}

type Detail struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s, details: %v", e.Status, e.Err, e.Details)
}

func (s *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message string   `json:"message"`
		Details []Detail `json:"details"`
	}{
		Message: s.Err.Error(),
		Details: s.Details,
	})
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
