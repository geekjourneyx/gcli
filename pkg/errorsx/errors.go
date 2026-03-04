package errorsx

import (
	"context"
	"errors"
	"fmt"
)

// Code is a stable machine-readable error code.
type Code string

const (
	CodeInputInvalid          Code = "INPUT_INVALID"
	CodeAuthMissingCreds      Code = "AUTH_MISSING_CREDENTIALS"
	CodeAuthRefreshFailed     Code = "AUTH_REFRESH_FAILED"
	CodeAuthScopeInsufficient Code = "AUTH_SCOPE_INSUFFICIENT"
	CodeAuthDeviceFlowFailed  Code = "AUTH_DEVICE_FLOW_FAILED"
	CodeAuthCodeFlowFailed    Code = "AUTH_CODE_FLOW_FAILED"
	CodeAuthStateMismatch     Code = "AUTH_STATE_MISMATCH"
	CodeAuthNoRefreshToken    Code = "AUTH_NO_REFRESH_TOKEN"
	CodeMailNotFound          Code = "MAIL_NOT_FOUND"
	CodeGmailAPIQuota         Code = "GMAIL_API_QUOTA"
	CodeGmailAPIUnavailable   Code = "GMAIL_API_UNAVAILABLE"
	CodeTimeout               Code = "TIMEOUT"
	CodeCanceled              Code = "CANCELED"
	CodeInternal              Code = "INTERNAL"
)

// AppError is the canonical error type for the CLI.
type AppError struct {
	Code      Code
	Message   string
	Retryable bool
	Details   map[string]string
	Err       error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code Code, message string, retryable bool) *AppError {
	return &AppError{Code: code, Message: message, Retryable: retryable}
}

func Wrap(code Code, message string, retryable bool, err error) *AppError {
	return &AppError{Code: code, Message: message, Retryable: retryable, Err: err}
}

// AddDetail appends a structured key/value detail to the error.
func (e *AppError) AddDetail(key, value string) *AppError {
	if e == nil {
		return nil
	}
	if key == "" {
		return e
	}
	if e.Details == nil {
		e.Details = map[string]string{}
	}
	e.Details[key] = value
	return e
}

func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return Wrap(CodeTimeout, "operation timed out", true, err)
	case errors.Is(err, context.Canceled):
		return Wrap(CodeCanceled, "operation canceled", false, err)
	default:
		return Wrap(CodeInternal, "unexpected internal error", false, err)
	}
}

func ExitCode(err error) int {
	appErr := From(err)
	switch appErr.Code {
	case CodeInputInvalid:
		return 2
	case CodeAuthMissingCreds, CodeAuthRefreshFailed, CodeAuthScopeInsufficient, CodeAuthDeviceFlowFailed, CodeAuthCodeFlowFailed, CodeAuthStateMismatch, CodeAuthNoRefreshToken:
		return 3
	case CodeMailNotFound:
		return 4
	case CodeGmailAPIQuota:
		return 5
	case CodeGmailAPIUnavailable:
		return 6
	case CodeTimeout:
		return 7
	case CodeCanceled:
		return 130
	default:
		return 1
	}
}
