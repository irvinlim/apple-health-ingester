package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

type Reason string

const (
	ReasonRetryableWrite Reason = "RetryableWrite"
	ReasonUnknown        Reason = "Unknown"
)

var (
	// baseError is used when there is no error being wrapped and the reason refers to the error itself.
	baseError = errors.New("base error")
)

// Error implements custom error types.
type Error interface {
	error
	GetReason() Reason
}

type wrappedError struct {
	error
	Reason  Reason
	Message string
}

var _ Error = (*wrappedError)(nil)

func (w *wrappedError) Error() string {
	if w.error == baseError {
		return w.error.Error()
	}
	return fmt.Sprintf("%v: %v", w.Message, w.error)
}

func (w *wrappedError) GetReason() Reason {
	return w.Reason
}

// NewRetryableWriteError returns a new RetryableWriteError.
func NewRetryableWriteError() error {
	return WrapRetryableWrite(baseError)
}

// WrapRetryableWrite wraps an error as a RetryableWriteError.
func WrapRetryableWrite(err error) error {
	return WrapfRetryableWrite(err, "")
}

// WrapfRetryableWrite wraps an error as a RetryableWriteError.
func WrapfRetryableWrite(err error, message string) error {
	return &wrappedError{
		error:   err,
		Reason:  ReasonRetryableWrite,
		Message: wrapMessage("temporary error writing to backend, will retry again later", message),
	}
}

// IsRetryableWrite tests if err is a RetryableWriteError.
func IsRetryableWrite(err error) bool {
	return GetReason(err) == ReasonRetryableWrite
}

func GetReason(err error) Reason {
	if wrappedErr := Error(nil); errors.As(err, &wrappedErr) {
		return wrappedErr.GetReason()
	}
	return ReasonUnknown
}

func wrapMessage(base string, message string) string {
	wrapped := base
	if len(message) > 0 {
		wrapped = fmt.Sprintf("%v: %v", message, wrapped)
	}
	return wrapped
}
