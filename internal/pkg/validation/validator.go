// Package validation provides a wrapper around the go-playground/validator library,
// adding support for thread-safe initialization and standardized error formatting.
//
// It allows for validating structs using tags and returning consistent errors
// that can be logged or returned to clients.
package validation

import (
	"errors"
	"fmt"
	"sync"

	gvalidator "github.com/go-playground/validator/v10"
)

// validator is a singleton instance of the go-playground validator.
var (
	validator         *gvalidator.Validate
	initValidatorOnce sync.Once
)

// ErrValidation is returned as the first error when validation fails.
// It acts as a high-level indicator that one or more validation rules were violated.
var ErrValidation = errors.New("validation error")

// errStringFormat defines the format for individual validation error messages.
const errStringFormat = "'%s': value '%v' does not meet the requirements for the '%s' validation"

// Init initializes the validator only once, enabling required field validation on structs.
//
// It is safe to call Init multiple times; only the first call will take effect.
func Init() {
	initValidatorOnce.Do(func() {
		validator = gvalidator.New(gvalidator.WithRequiredStructEnabled())
	})
}

// formatError takes a validator error and transforms it into a detailed, multi-error
// chain with human-readable messages. The first error in the chain is always ErrValidation.
//
// Each field error is formatted using errStringFormat.
func formatError(err error) error {
	var validationErrors gvalidator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}

	errs := []error{ErrValidation}
	for _, validationErr := range validationErrors {
		var (
			field = validationErr.Field()
			tag   = validationErr.Tag()
			value = validationErr.Value()
			err   = fmt.Errorf(errStringFormat, field, value, tag)
		)

		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// Validate validates a struct using the singleton validator instance.
//
// It returns nil if the struct passes validation, or an error containing all violations
// if validation fails. You must call Init() before using this function.
func Validate(v any) error {
	if err := validator.Struct(v); err != nil {
		return formatError(err)
	}

	return nil
}
