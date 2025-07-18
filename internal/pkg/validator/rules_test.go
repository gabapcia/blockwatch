package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredAloneValidator(t *testing.T) {
	t.Run("valid - first field has value, others empty", func(t *testing.T) {
		input := struct {
			Field1 string `validate:"required_alone"`
			Field2 string `validate:"required_alone"`
			Field3 string `validate:"required_alone"`
		}{
			Field1: "value",
			Field2: "",
			Field3: "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - second field has value, others empty", func(t *testing.T) {
		input := struct {
			Field1 string `validate:"required_alone"`
			Field2 string `validate:"required_alone"`
			Field3 string `validate:"required_alone"`
		}{
			Field1: "",
			Field2: "value",
			Field3: "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("invalid - multiple fields have values", func(t *testing.T) {
		input := struct {
			Field1 string `validate:"required_alone"`
			Field2 string `validate:"required_alone"`
			Field3 string `validate:"required_alone"`
		}{
			Field1: "value1",
			Field2: "value2",
			Field3: "",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("valid - all fields empty", func(t *testing.T) {
		input := struct {
			Field1 string `validate:"required_alone"`
			Field2 string `validate:"required_alone"`
			Field3 string `validate:"required_alone"`
		}{
			Field1: "",
			Field2: "",
			Field3: "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - works with different types", func(t *testing.T) {
		input := struct {
			StringField string `validate:"required_alone"`
			IntField    int    `validate:"required_alone"`
			BoolField   bool   `validate:"required_alone"`
		}{
			StringField: "",
			IntField:    42,
			BoolField:   false,
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - all different types empty", func(t *testing.T) {
		input := struct {
			StringField string `validate:"required_alone"`
			IntField    int    `validate:"required_alone"`
			BoolField   bool   `validate:"required_alone"`
		}{
			StringField: "",
			IntField:    0,
			BoolField:   false,
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - bool field true", func(t *testing.T) {
		input := struct {
			StringField string `validate:"required_alone"`
			IntField    int    `validate:"required_alone"`
			BoolField   bool   `validate:"required_alone"`
		}{
			StringField: "",
			IntField:    0,
			BoolField:   true,
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("invalid - mixed validation tags with multiple values", func(t *testing.T) {
		input := struct {
			Field1 string `validate:"required_alone"`
			Field2 string `validate:"required"`
			Field3 string `validate:"required_alone"`
		}{
			Field1: "value1",
			Field2: "required_value",
			Field3: "",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("valid - single field with required_alone", func(t *testing.T) {
		input := struct {
			OnlyField string `validate:"required_alone"`
		}{
			OnlyField: "value",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - single field empty with required_alone", func(t *testing.T) {
		input := struct {
			OnlyField string `validate:"required_alone"`
		}{
			OnlyField: "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("invalid - required_alone field with value and other non-required_alone field with value", func(t *testing.T) {
		input := struct {
			Field1 string `validate:"required_alone"`
			Field2 string // no validation tag
			Field3 string `validate:"required_alone"`
		}{
			Field1: "value1",
			Field2: "value2", // this field has no validation tag but has a value
			Field3: "",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("valid - struct field with required_alone has value, others empty", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		input := struct {
			StructField NestedStruct `validate:"required_alone"`
			StringField string       `validate:"required_alone"`
			IntField    int          `validate:"required_alone"`
		}{
			StructField: NestedStruct{Name: "test"},
			StringField: "",
			IntField:    0,
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - all struct fields empty", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		input := struct {
			StructField1 NestedStruct `validate:"required_alone"`
			StructField2 NestedStruct `validate:"required_alone"`
			StringField  string       `validate:"required_alone"`
		}{
			StructField1: NestedStruct{},
			StructField2: NestedStruct{},
			StringField:  "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("invalid - multiple struct fields have values", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		input := struct {
			StructField1 NestedStruct `validate:"required_alone"`
			StructField2 NestedStruct `validate:"required_alone"`
			StringField  string       `validate:"required_alone"`
		}{
			StructField1: NestedStruct{Name: "test1"},
			StructField2: NestedStruct{Name: "test2"},
			StringField:  "",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("valid - pointer field with required_alone has value, others empty", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		value := &NestedStruct{Name: "test"}
		input := struct {
			PointerField *NestedStruct `validate:"required_alone"`
			StringField  string        `validate:"required_alone"`
			IntField     int           `validate:"required_alone"`
		}{
			PointerField: value,
			StringField:  "",
			IntField:     0,
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - all pointer fields nil (using omitempty)", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		input := struct {
			PointerField1 *NestedStruct `validate:"omitempty,required_alone"`
			PointerField2 *NestedStruct `validate:"omitempty,required_alone"`
			StringField   string        `validate:"required_alone"`
		}{
			PointerField1: nil,
			PointerField2: nil,
			StringField:   "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("invalid - multiple pointer fields have values", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		value1 := &NestedStruct{Name: "test1"}
		value2 := &NestedStruct{Name: "test2"}
		input := struct {
			PointerField1 *NestedStruct `validate:"omitempty,required_alone"`
			PointerField2 *NestedStruct `validate:"omitempty,required_alone"`
			StringField   string        `validate:"required_alone"`
		}{
			PointerField1: value1,
			PointerField2: value2,
			StringField:   "",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("invalid - pointer and struct fields both have values", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		value := &NestedStruct{Name: "test"}
		input := struct {
			PointerField *NestedStruct `validate:"omitempty,required_alone"`
			StructField  NestedStruct  `validate:"required_alone"`
			StringField  string        `validate:"required_alone"`
		}{
			PointerField: value,
			StructField:  NestedStruct{Name: "test2"},
			StringField:  "",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("valid - pointer to empty struct", func(t *testing.T) {
		type NestedStruct struct {
			Name string
		}

		value := &NestedStruct{} // pointer to empty struct
		input := struct {
			PointerField *NestedStruct `validate:"omitempty,required_alone"`
			StringField  string        `validate:"required_alone"`
		}{
			PointerField: value,
			StringField:  "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("valid - struct with unexported fields", func(t *testing.T) {
		input := struct {
			PublicField  string `validate:"required_alone"`
			privateField string // unexported field should be ignored
			AnotherField string `validate:"required_alone"`
		}{
			PublicField:  "value",
			privateField: "ignored", // this should be ignored by the validator
			AnotherField: "",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("edge case - deeply nested struct", func(t *testing.T) {
		type Level3 struct {
			Field1 string `validate:"required_alone"`
			Field2 string `validate:"required_alone"`
		}

		type Level2 struct {
			Nested Level3 `validate:"required"`
		}

		type Level1 struct {
			Nested Level2 `validate:"required"`
		}

		input := Level1{
			Nested: Level2{
				Nested: Level3{
					Field1: "value",
					Field2: "",
				},
			},
		}

		err := Validate(input)
		assert.NoError(t, err)
	})
}
