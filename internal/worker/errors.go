package worker

import (
	"errors"

	"go.temporal.io/sdk/temporal"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
)

// Unwraps the application error from temporal into a seyerr if possible.
//
// Returns true if the error is convertible to a seymour error.
// Returns false otherwise.
func asSeyerr(err error, seyerr **seyerrs.Error) bool {
	for {
		appErr := &temporal.ApplicationError{}
		if !errors.As(err, &appErr) {
			return false
		}

		if appErr.Type() != "seyerr" {
			err = errors.Unwrap(err) // Go one level deeper
			continue
		}

		ourErr := &seyerrs.Error{}
		if err := appErr.Details(&ourErr); err != nil {
			return false
		}

		*seyerr = ourErr
		break
	}

	return true
}

// Turns an error from a app call into a temporal application error.
//
// Later can be used with asSeyerr to unwrap the error with the ergo
// of an actual seymour error.
func appErr(err error) error {
	if err == nil {
		return nil
	}

	seyerr := &seyerrs.Error{}
	if errors.As(err, &seyerr) {
		return temporal.NewApplicationError(seyerr.Error(), "seyerr", seyerr)
	}

	return err
}
