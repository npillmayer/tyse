package core

import (
	"errors"
	"fmt"
	"os"
)

// General error codes
const (
	NOERROR     int = 0
	EMISSING    int = 122 // resource does not exist
	EINVALID    int = 123 // validation failed
	ECONNECTION int = 124 // remote resource not connected
	EINTERNAL   int = 125 // internal error
)

func errorText(ecode int) string {
	switch ecode {
	case NOERROR:
		return "OK"
	case EMISSING:
		return "not found"
	case EINVALID:
		return "invalid"
	case ECONNECTION:
		return "transmission-error"
	case EINTERNAL:
		return "internal error"
	}
	return "undefined error"
}

// AppError is an error with an associated error code and a user-message.
type AppError interface {
	error
	ErrorCode() int
	UserMessage() string
}

type coreError struct {
	error
	code int
	msg  string
}

func (e coreError) Unwrap() error {
	return e.error
}

func (e coreError) Error() string {
	return fmt.Sprintf("[%d] %v", e.code, e.error)
}

func (e coreError) ErrorCode() int {
	return e.code
}

func (e coreError) UserMessage() string {
	return e.msg
}

var _ AppError = coreError{}

// ErrorWithCode adds an error code to err's error chain.
// Unlike pkg/errors, ErrorWithCode will wrap nil error.
func ErrorWithCode(err error, code int) error {
	if err == nil {
		err = errors.New(errorText(code))
	}
	return coreError{err, code, errorText(code)}
}

// WrapError wraps an error in a core error, featuring an error code and
// a user message.
// If err is nil, an error denoting NOERROR is returned.
func WrapError(err error, code int, format string, v ...interface{}) error {
	if err == nil {
		err = errors.New(errorText(code))
	}
	msg := fmt.Sprintf(format, v...)
	return coreError{err, code, msg}
}

// Code returns the status code associated with an error.
// If no status code is found, it returns EINTERNAL.
// If err is nil, NOERROR is returned.
func Code(err error) (code int) {
	if err == nil {
		return NOERROR
	}
	if e := AppError(nil); errors.As(err, &e) {
		return e.ErrorCode()
	}
	return EINTERNAL
}

// UserMessage returns the user message associated with an error.
// If no message is found, it checks StatusCode and returns that message.
// If err is nil, it returns "".
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	if e := AppError(nil); errors.As(err, &e) {
		return e.UserMessage()
	}
	return errorText(Code(err))
}

// Error creates an error with an error code and a user-message.
func Error(code int, format string, v ...interface{}) error {
	return coreError{
		errors.New(errorText(code)),
		code,
		fmt.Sprintf(format, v...),
	}
}

func UserError(err error) {
	if e, ok := err.(AppError); ok {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", e.ErrorCode(), e.UserMessage())
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
}
