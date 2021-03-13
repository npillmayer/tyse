package core

import (
	"errors"
	"fmt"
)

// General error codes
const (
	NOERROR   int = iota
	EMISSING      // resource does not exist
	EINVALID      // validation failed
	EINTERNAL     // internal error
)

func errorText(ecode int) string {
	switch ecode {
	case NOERROR:
		return "OK"
	case EMISSING:
		return "not found"
	case EINVALID:
		return "invalid"
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

func (e coreError) StatusCode() int {
	return e.code
}

func (e coreError) UserMessage() string {
	return e.msg
}

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
