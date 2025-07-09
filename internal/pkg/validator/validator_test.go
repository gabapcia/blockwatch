package validator

import (
	"errors"
	"testing"

	gvalidator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorInitialization(t *testing.T) {
	t.Run("should initialize validator instance", func(t *testing.T) {
		assert.NotNil(t, validator)
	})

	t.Run("should work correctly after initialization", func(t *testing.T) {
		type SimpleStruct struct {
			Name string `validate:"required"`
		}

		validStruct := SimpleStruct{Name: "test"}

		err := validator.Struct(validStruct)
		assert.NoError(t, err)
	})

	t.Run("should support required struct validation", func(t *testing.T) {
		type NestedStruct struct {
			Inner struct {
				Value string `validate:"required"`
			} `validate:"required"`
		}

		nested := NestedStruct{}
		nested.Inner.Value = "test"

		err := validator.Struct(nested)
		assert.NoError(t, err)
	})
}

func TestFormatError(t *testing.T) {
	t.Run("should transform validation errors to formatted errors", func(t *testing.T) {
		testValidator := gvalidator.New()

		type TestStruct struct {
			Name string `validate:"required"`
		}

		testStruct := TestStruct{Name: ""}

		err := testValidator.Struct(testStruct)
		require.Error(t, err)

		formattedErr := formatError(err)

		assert.ErrorIs(t, formattedErr, ErrValidationFailed)
		assert.Contains(t, formattedErr.Error(), "'Name': value '' does not meet the requirements for the 'required' validation")
	})

	t.Run("should return original error when not validation error", func(t *testing.T) {
		originalErr := errors.New("database connection failed")
		formattedErr := formatError(originalErr)

		assert.Equal(t, originalErr, formattedErr)
	})

	t.Run("should handle multiple validation errors", func(t *testing.T) {
		testValidator := gvalidator.New()

		type MultiFieldStruct struct {
			Name  string `validate:"required"`
			Email string `validate:"required,email"`
		}

		testStruct := MultiFieldStruct{
			Name:  "",
			Email: "invalid",
		}

		err := testValidator.Struct(testStruct)
		require.Error(t, err)

		formattedErr := formatError(err)

		assert.ErrorIs(t, formattedErr, ErrValidationFailed)
		errStr := formattedErr.Error()
		assert.Contains(t, errStr, "'Name': value '' does not meet the requirements for the 'required' validation")
		assert.Contains(t, errStr, "'Email': value 'invalid' does not meet the requirements for the 'email' validation")
	})
}

func TestValidate(t *testing.T) {
	t.Run("should pass when all required fields are present", func(t *testing.T) {
		type User struct {
			Name  string `validate:"required"`
			Email string `validate:"required,email"`
			Age   int    `validate:"min=0,max=150"`
		}

		user := User{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   25,
		}

		err := Validate(user)
		assert.NoError(t, err)
	})

	t.Run("should pass when using boundary values", func(t *testing.T) {
		type Product struct {
			Name     string `validate:"required"`
			Price    int    `validate:"min=0"`
			Quantity int    `validate:"min=0,max=1000"`
		}

		product := Product{
			Name:     "Widget",
			Price:    0,
			Quantity: 1000,
		}

		err := Validate(product)
		assert.NoError(t, err)
	})

	t.Run("should pass when validating empty struct", func(t *testing.T) {
		type EmptyStruct struct{}

		empty := EmptyStruct{}

		err := Validate(empty)
		assert.NoError(t, err)
	})

	t.Run("should pass when enum value is valid", func(t *testing.T) {
		type Order struct {
			ID     string `validate:"required"`
			Status string `validate:"required,oneof=pending processing shipped delivered cancelled"`
		}

		order := Order{
			ID:     "order-123",
			Status: "processing",
		}

		err := Validate(order)
		assert.NoError(t, err)
	})

	t.Run("should pass when validating nested struct", func(t *testing.T) {
		type Address struct {
			Street string `validate:"required"`
			City   string `validate:"required"`
		}

		type Customer struct {
			Name    string  `validate:"required"`
			Address Address `validate:"required"`
		}

		customer := Customer{
			Name: "Jane Smith",
			Address: Address{
				Street: "123 Main St",
				City:   "Springfield",
			},
		}

		err := Validate(customer)
		assert.NoError(t, err)
	})

	t.Run("should fail when required field is empty", func(t *testing.T) {
		type User struct {
			Name  string `validate:"required"`
			Email string `validate:"required,email"`
		}

		user := User{
			Name:  "",
			Email: "john@example.com",
		}

		err := Validate(user)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "'Name': value '' does not meet the requirements for the 'required' validation")
	})

	t.Run("should fail when email format is invalid", func(t *testing.T) {
		type Contact struct {
			Email string `validate:"required,email"`
		}

		contact := Contact{
			Email: "not-an-email",
		}

		err := Validate(contact)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "'Email': value 'not-an-email' does not meet the requirements for the 'email' validation")
	})

	t.Run("should fail when numeric value is below minimum", func(t *testing.T) {
		type Product struct {
			Price int `validate:"min=0"`
		}

		product := Product{
			Price: -10,
		}

		err := Validate(product)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "'Price': value '-10' does not meet the requirements for the 'min' validation")
	})

	t.Run("should fail when numeric value is above maximum", func(t *testing.T) {
		type Person struct {
			Age int `validate:"max=150"`
		}

		person := Person{
			Age: 200,
		}

		err := Validate(person)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "'Age': value '200' does not meet the requirements for the 'max' validation")
	})

	t.Run("should fail when enum value is invalid", func(t *testing.T) {
		type Task struct {
			Status string `validate:"required,oneof=todo in_progress done"`
		}

		task := Task{
			Status: "invalid_status",
		}

		err := Validate(task)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "'Status': value 'invalid_status' does not meet the requirements for the 'oneof' validation")
	})

	t.Run("should fail with multiple validation errors", func(t *testing.T) {
		type Registration struct {
			Username string `validate:"required"`
			Email    string `validate:"required,email"`
			Age      int    `validate:"min=18,max=100"`
			Role     string `validate:"required,oneof=admin user guest"`
		}

		registration := Registration{
			Username: "",
			Email:    "invalid-email",
			Age:      15,
			Role:     "superuser",
		}

		err := Validate(registration)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)

		errStr := err.Error()
		assert.Contains(t, errStr, "'Username': value '' does not meet the requirements for the 'required' validation")
		assert.Contains(t, errStr, "'Email': value 'invalid-email' does not meet the requirements for the 'email' validation")
		assert.Contains(t, errStr, "'Age': value '15' does not meet the requirements for the 'min' validation")
		assert.Contains(t, errStr, "'Role': value 'superuser' does not meet the requirements for the 'oneof' validation")
	})

	t.Run("should fail when nested struct validation fails", func(t *testing.T) {
		type Address struct {
			Street string `validate:"required"`
			City   string `validate:"required"`
		}

		type Customer struct {
			Name    string  `validate:"required"`
			Address Address `validate:"required"`
		}

		customer := Customer{
			Name: "John Doe",
			Address: Address{
				Street: "",
				City:   "Springfield",
			},
		}

		err := Validate(customer)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("should fail when input is not struct", func(t *testing.T) {
		testCases := []interface{}{
			"test string",
			42,
			nil,
			[]string{"test"},
			map[string]string{"key": "value"},
		}

		for _, input := range testCases {
			err := Validate(input)
			assert.Error(t, err)
		}
	})

	t.Run("should validate multiple enum options", func(t *testing.T) {
		type Priority struct {
			Level string `validate:"required,oneof=low medium high critical"`
		}

		validPriorities := []string{"low", "medium", "high", "critical"}
		for _, priority := range validPriorities {
			p := Priority{Level: priority}
			err := Validate(p)
			assert.NoError(t, err, "priority %s should be valid", priority)
		}

		invalidPriority := Priority{Level: "urgent"}
		err := Validate(invalidPriority)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "'Level': value 'urgent' does not meet the requirements for the 'oneof' validation")
	})
}
