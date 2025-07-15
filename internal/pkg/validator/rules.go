package validator

import (
	"reflect"

	gvalidator "github.com/go-playground/validator/v10"
)

// requiredAloneValidator validates that if a field with "required_alone" tag has a value,
// then all other fields in the struct must be empty.
//
// This validator works similarly to "required_without_all" but automatically considers all other fields
// in the struct instead of requiring the user to specify which fields to compare against.
//
// The field validation passes if:
// 1. The current field is empty (no restriction on other fields), OR
// 2. The current field has a value AND all other fields in the struct are empty
//
// The field validation fails if:
// 1. The current field has a value AND any other field in the struct also has a value
//
// This ensures that the field marked with "required_alone" can only have a value if it's the only field with a value.
//
// IMPORTANT: Due to go-playground/validator library limitations, nil pointers fail validation before
// this custom validator is called. For pointer fields, use "omitempty,required_alone" to properly
// handle nil values: `validate:"omitempty,required_alone"`
func requiredAloneValidator(fl gvalidator.FieldLevel) bool {
	// Get the current field value
	currentFieldHasValue := !fl.Field().IsZero()

	// If the current field is empty, validation always passes
	if !currentFieldHasValue {
		return true
	}

	// Get the parent struct
	parent := fl.Parent()
	if parent.Kind() != reflect.Struct {
		return false
	}

	// If current field has a value, check that all other fields are empty
	structType := parent.Type()
	for i := 0; i < parent.NumField(); i++ {
		field := parent.Field(i)
		fieldType := structType.Field(i)

		// Skip the current field being validated
		if fieldType.Name == fl.FieldName() {
			continue
		}

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// If any other field has a value, validation fails
		if !field.IsZero() {
			return false
		}
	}

	// Current field has value and all other fields are empty
	return true
}
