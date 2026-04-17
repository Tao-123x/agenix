package agenix

import "fmt"

const (
	ErrInvalidInput       = "InvalidInput"
	ErrUnsupportedAdapter = "UnsupportedAdapter"
	ErrPermissionDenied   = "PermissionDenied"
	ErrNotFound           = "NotFound"
	ErrTimeout            = "Timeout"
	ErrDriverError        = "DriverError"
	ErrPolicyViolation    = "PolicyViolation"
	ErrVerificationFailed = "VerificationFailed"
)

type Error struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Class, e.Message)
}

func NewError(class, message string) error {
	return Error{Class: class, Message: message}
}

func WrapError(class, message string, err error) error {
	if err == nil {
		return NewError(class, message)
	}
	return Error{Class: class, Message: fmt.Sprintf("%s: %v", message, err)}
}

func IsErrorClass(err error, class string) bool {
	if err == nil {
		return false
	}
	if agenixErr, ok := err.(Error); ok {
		return agenixErr.Class == class
	}
	if agenixErr, ok := err.(*Error); ok {
		return agenixErr.Class == class
	}
	return false
}

func ErrorClass(err error) string {
	if err == nil {
		return ""
	}
	if agenixErr, ok := err.(Error); ok {
		return agenixErr.Class
	}
	if agenixErr, ok := err.(*Error); ok {
		return agenixErr.Class
	}
	return ErrDriverError
}
