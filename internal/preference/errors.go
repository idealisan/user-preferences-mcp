package preference

import (
	"fmt"
)

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

func DatabaseError(err error) error {
	return &CodedError{Code: "DATABASE_ERROR", Message: err.Error()}
}
