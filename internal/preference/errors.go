package preference

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("preference not found")

type CodedError struct {
	Code    string
	Message string
}

func (e *CodedError) Error() string {
	return e.Code + ": " + e.Message
}

func InvalidArgument(format string, args ...any) error {
	return &CodedError{Code: "INVALID_ARGUMENT", Message: fmt.Sprintf(format, args...)}
}

func InvalidJSON(format string, args ...any) error {
	return &CodedError{Code: "INVALID_JSON", Message: fmt.Sprintf(format, args...)}
}

func InvalidScope(format string, args ...any) error {
	return &CodedError{Code: "INVALID_SCOPE", Message: fmt.Sprintf(format, args...)}
}

func DatabaseError(err error) error {
	return &CodedError{Code: "DATABASE_ERROR", Message: err.Error()}
}
