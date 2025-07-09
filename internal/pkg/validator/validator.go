// Package validator provides a thin wrapper around the go-playground/validator library,
// enabling declarative struct validation with standardized error formatting.
//
// It supports validating struct fields using tags (e.g., `validate:"required"`) and returns
// descriptive error messages when validation rules are violated. This package is initialized
// automatically and safe to use directly.
package validator

import (
	"errors"
	"fmt"

	gvalidator "github.com/go-playground/validator/v10"
)

// ErrValidationFailed is returned as the first error in a multi-error chain when validation fails.
//
// This sentinel error allows callers to detect validation failures explicitly,
// even when multiple field errors are returned.
var ErrValidationFailed = errors.New("struct validation failed")

// validator is a singleton instance of the go-playground validator,
// initialized automatically on package load.
var validator *gvalidator.Validate

// errStringFormat defines the template used to describe individual validation errors.
//
// Example: "'Address': value '0x' does not meet the requirements for the 'required' validation"
const errStringFormat = "'%s': value '%v' does not meet the requirements for the '%s' validation"

// init initializes the singleton validator instance automatically on package import.
//
// It enables validation for required fields in structs using tags like `validate:"required"`.
func init() {
	validator = gvalidator.New(gvalidator.WithRequiredStructEnabled())
}

// formatError transforms a raw validator error into a structured, human-readable multi-error chain.
//
// If the input is a set of validation errors, it returns a combined error with ErrValidationFailed as the root,
// followed by a formatted message for each field error. Otherwise, the original error is returned unchanged.
func formatError(err error) error {
	var validationErrors gvalidator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}

	errs := []error{ErrValidationFailed}
	for _, validationErr := range validationErrors {
		err := fmt.Errorf(errStringFormat,
			validationErr.Field(),
			validationErr.Value(),
			validationErr.Tag(),
		)

		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// Validate checks if the given struct satisfies its validation tags.
//
// It returns nil if all fields pass validation. Otherwise, it returns a combined error that includes
// ErrValidationFailed and one formatted message for each field that failed validation.
//
// Example usage:
//
//	type Input struct {
//	    Name string `validate:"required"`
//	}
//
//	if err := validator.Validate(input); errors.Is(err, validator.ErrValidationFailed) {
//	    // Handle validation failure
//	}
func Validate(v any) error {
	if err := validator.Struct(v); err != nil {
		return formatError(err)
	}

	return nil
}
