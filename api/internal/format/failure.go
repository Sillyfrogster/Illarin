package format

import (
	"errors"
	"fmt"
)

type FailureReason string

const (
	FailureMalformedInput     FailureReason = "malformed_input"
	FailureUnsupportedFormat  FailureReason = "unsupported_format"
	FailureUnsupportedVersion FailureReason = "unsupported_version"
	FailureSafetyViolation    FailureReason = "safety_violation"
	FailureWrongKind          FailureReason = "wrong_kind"
	FailureLimitExceeded      FailureReason = "limit_exceeded"
	FailureInternal           FailureReason = "internal_failure"
)

type failure struct {
	reason FailureReason
	cause  error
}

func (f failure) Error() string { return fmt.Sprintf("%s: %v", f.reason, f.cause) }
func (f failure) Unwrap() error { return f.cause }

func UnsupportedVersion(err error) error {
	return failure{reason: FailureUnsupportedVersion, cause: err}
}

func MalformedInput(err error) error {
	return failure{reason: FailureMalformedInput, cause: err}
}

func SafetyViolation(err error) error {
	return failure{reason: FailureSafetyViolation, cause: err}
}

func LimitExceeded(err error) error {
	return failure{reason: FailureLimitExceeded, cause: err}
}

func InternalFailure(err error) error {
	return failure{reason: FailureInternal, cause: err}
}

func FailureOf(err error) (FailureReason, bool) {
	var classified failure
	if !errors.As(err, &classified) {
		return "", false
	}
	return classified.reason, true
}
