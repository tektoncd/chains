// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/config/cluster/v3/circuit_breaker.proto

package clusterv3

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"

	v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort

	_ = v3.RoutingPriority(0)
)

// Validate checks the field values on CircuitBreakers with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *CircuitBreakers) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on CircuitBreakers with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// CircuitBreakersMultiError, or nil if none found.
func (m *CircuitBreakers) ValidateAll() error {
	return m.validate(true)
}

func (m *CircuitBreakers) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetThresholds() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, CircuitBreakersValidationError{
						field:  fmt.Sprintf("Thresholds[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, CircuitBreakersValidationError{
						field:  fmt.Sprintf("Thresholds[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return CircuitBreakersValidationError{
					field:  fmt.Sprintf("Thresholds[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	for idx, item := range m.GetPerHostThresholds() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, CircuitBreakersValidationError{
						field:  fmt.Sprintf("PerHostThresholds[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, CircuitBreakersValidationError{
						field:  fmt.Sprintf("PerHostThresholds[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return CircuitBreakersValidationError{
					field:  fmt.Sprintf("PerHostThresholds[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return CircuitBreakersMultiError(errors)
	}

	return nil
}

// CircuitBreakersMultiError is an error wrapping multiple validation errors
// returned by CircuitBreakers.ValidateAll() if the designated constraints
// aren't met.
type CircuitBreakersMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m CircuitBreakersMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m CircuitBreakersMultiError) AllErrors() []error { return m }

// CircuitBreakersValidationError is the validation error returned by
// CircuitBreakers.Validate if the designated constraints aren't met.
type CircuitBreakersValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CircuitBreakersValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CircuitBreakersValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CircuitBreakersValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CircuitBreakersValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CircuitBreakersValidationError) ErrorName() string { return "CircuitBreakersValidationError" }

// Error satisfies the builtin error interface
func (e CircuitBreakersValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCircuitBreakers.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CircuitBreakersValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CircuitBreakersValidationError{}

// Validate checks the field values on CircuitBreakers_Thresholds with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *CircuitBreakers_Thresholds) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on CircuitBreakers_Thresholds with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// CircuitBreakers_ThresholdsMultiError, or nil if none found.
func (m *CircuitBreakers_Thresholds) ValidateAll() error {
	return m.validate(true)
}

func (m *CircuitBreakers_Thresholds) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if _, ok := v3.RoutingPriority_name[int32(m.GetPriority())]; !ok {
		err := CircuitBreakers_ThresholdsValidationError{
			field:  "Priority",
			reason: "value must be one of the defined enum values",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetMaxConnections()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxConnections",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxConnections",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMaxConnections()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_ThresholdsValidationError{
				field:  "MaxConnections",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetMaxPendingRequests()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxPendingRequests",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxPendingRequests",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMaxPendingRequests()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_ThresholdsValidationError{
				field:  "MaxPendingRequests",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetMaxRequests()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxRequests",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxRequests",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMaxRequests()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_ThresholdsValidationError{
				field:  "MaxRequests",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetMaxRetries()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxRetries",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxRetries",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMaxRetries()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_ThresholdsValidationError{
				field:  "MaxRetries",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetRetryBudget()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "RetryBudget",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "RetryBudget",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetRetryBudget()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_ThresholdsValidationError{
				field:  "RetryBudget",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for TrackRemaining

	if all {
		switch v := interface{}(m.GetMaxConnectionPools()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxConnectionPools",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_ThresholdsValidationError{
					field:  "MaxConnectionPools",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMaxConnectionPools()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_ThresholdsValidationError{
				field:  "MaxConnectionPools",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return CircuitBreakers_ThresholdsMultiError(errors)
	}

	return nil
}

// CircuitBreakers_ThresholdsMultiError is an error wrapping multiple
// validation errors returned by CircuitBreakers_Thresholds.ValidateAll() if
// the designated constraints aren't met.
type CircuitBreakers_ThresholdsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m CircuitBreakers_ThresholdsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m CircuitBreakers_ThresholdsMultiError) AllErrors() []error { return m }

// CircuitBreakers_ThresholdsValidationError is the validation error returned
// by CircuitBreakers_Thresholds.Validate if the designated constraints aren't met.
type CircuitBreakers_ThresholdsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CircuitBreakers_ThresholdsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CircuitBreakers_ThresholdsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CircuitBreakers_ThresholdsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CircuitBreakers_ThresholdsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CircuitBreakers_ThresholdsValidationError) ErrorName() string {
	return "CircuitBreakers_ThresholdsValidationError"
}

// Error satisfies the builtin error interface
func (e CircuitBreakers_ThresholdsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCircuitBreakers_Thresholds.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CircuitBreakers_ThresholdsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CircuitBreakers_ThresholdsValidationError{}

// Validate checks the field values on CircuitBreakers_Thresholds_RetryBudget
// with the rules defined in the proto definition for this message. If any
// rules are violated, the first error encountered is returned, or nil if
// there are no violations.
func (m *CircuitBreakers_Thresholds_RetryBudget) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on
// CircuitBreakers_Thresholds_RetryBudget with the rules defined in the proto
// definition for this message. If any rules are violated, the result is a
// list of violation errors wrapped in
// CircuitBreakers_Thresholds_RetryBudgetMultiError, or nil if none found.
func (m *CircuitBreakers_Thresholds_RetryBudget) ValidateAll() error {
	return m.validate(true)
}

func (m *CircuitBreakers_Thresholds_RetryBudget) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetBudgetPercent()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_Thresholds_RetryBudgetValidationError{
					field:  "BudgetPercent",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_Thresholds_RetryBudgetValidationError{
					field:  "BudgetPercent",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetBudgetPercent()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_Thresholds_RetryBudgetValidationError{
				field:  "BudgetPercent",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetMinRetryConcurrency()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, CircuitBreakers_Thresholds_RetryBudgetValidationError{
					field:  "MinRetryConcurrency",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, CircuitBreakers_Thresholds_RetryBudgetValidationError{
					field:  "MinRetryConcurrency",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMinRetryConcurrency()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CircuitBreakers_Thresholds_RetryBudgetValidationError{
				field:  "MinRetryConcurrency",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return CircuitBreakers_Thresholds_RetryBudgetMultiError(errors)
	}

	return nil
}

// CircuitBreakers_Thresholds_RetryBudgetMultiError is an error wrapping
// multiple validation errors returned by
// CircuitBreakers_Thresholds_RetryBudget.ValidateAll() if the designated
// constraints aren't met.
type CircuitBreakers_Thresholds_RetryBudgetMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m CircuitBreakers_Thresholds_RetryBudgetMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m CircuitBreakers_Thresholds_RetryBudgetMultiError) AllErrors() []error { return m }

// CircuitBreakers_Thresholds_RetryBudgetValidationError is the validation
// error returned by CircuitBreakers_Thresholds_RetryBudget.Validate if the
// designated constraints aren't met.
type CircuitBreakers_Thresholds_RetryBudgetValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CircuitBreakers_Thresholds_RetryBudgetValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CircuitBreakers_Thresholds_RetryBudgetValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CircuitBreakers_Thresholds_RetryBudgetValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CircuitBreakers_Thresholds_RetryBudgetValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CircuitBreakers_Thresholds_RetryBudgetValidationError) ErrorName() string {
	return "CircuitBreakers_Thresholds_RetryBudgetValidationError"
}

// Error satisfies the builtin error interface
func (e CircuitBreakers_Thresholds_RetryBudgetValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCircuitBreakers_Thresholds_RetryBudget.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CircuitBreakers_Thresholds_RetryBudgetValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CircuitBreakers_Thresholds_RetryBudgetValidationError{}