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

	t.Run("valid - object composition with embedded structs", func(t *testing.T) {
		type RandomStruct struct {
			RandomField string `validate:"required"`
		}

		type RandomInlineStruct struct {
			Field1 *RandomStruct `validate:"omitempty,required_alone"`
			Field2 *RandomStruct `validate:"omitempty,required_alone"`
		}

		type Input struct {
			RandomInlineStruct `validate:"required"`
			Field3             string `validate:"required"`
		}

		input := Input{
			RandomInlineStruct: RandomInlineStruct{
				Field1: &RandomStruct{RandomField: "test"},
				Field2: nil,
			},
			Field3: "random-key",
		}

		err := Validate(input)
		assert.NoError(t, err)
	})

	t.Run("invalid - object composition error contains required_alone tag", func(t *testing.T) {
		type ConfigStruct struct {
			Setting string `validate:"required"`
		}

		type ComposedStruct struct {
			Config1 *ConfigStruct `validate:"omitempty,required_alone"`
			Config2 *ConfigStruct `validate:"omitempty,required_alone"`
		}

		// Both embedded structs have values - should fail with required_alone in error
		input := struct {
			ComposedStruct `validate:"required"`
			APIKey         string `validate:"required"`
		}{
			ComposedStruct: ComposedStruct{
				Config1: &ConfigStruct{Setting: "value1"}, // has value
				Config2: &ConfigStruct{Setting: "value2"}, // also has value - violation
			},
			APIKey: "api-key",
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "required_alone")
	})

	t.Run("invalid - composition error message contains field names and required_alone tag", func(t *testing.T) {
		type DatabaseConfig struct {
			Host string `validate:"required"`
			Port int    `validate:"required"`
		}

		type RedisConfig struct {
			URL      string `validate:"required"`
			Password string `validate:"required"`
		}

		// Both composed objects have values - should fail required_alone
		input := struct {
			Database DatabaseConfig `validate:"required_alone"`
			Redis    RedisConfig    `validate:"required_alone"`
			APIKey   string         `validate:"required_alone"`
		}{
			Database: DatabaseConfig{Host: "localhost", Port: 5432},             // has value
			Redis:    RedisConfig{URL: "redis://localhost", Password: "secret"}, // also has value - violation
			APIKey:   "",                                                        // empty
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "required_alone")
		assert.Contains(t, err.Error(), "Database")
		assert.Contains(t, err.Error(), "Redis")
	})

	t.Run("invalid - nested composition error contains required_alone validation tag", func(t *testing.T) {
		type InnerConfig struct {
			Value string `validate:"required"`
		}

		type MiddleConfig struct {
			Inner1 InnerConfig `validate:"required_alone"`
			Inner2 InnerConfig `validate:"required_alone"`
		}

		type OuterConfig struct {
			Middle MiddleConfig `validate:"required_alone"`
			Direct string       `validate:"required_alone"`
		}

		// Nested violation: Inner1 and Inner2 both have values
		input := OuterConfig{
			Middle: MiddleConfig{
				Inner1: InnerConfig{Value: "value1"}, // has value
				Inner2: InnerConfig{Value: "value2"}, // also has value - violation
			},
			Direct: "", // empty
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "required_alone")
		assert.Contains(t, err.Error(), "Inner1")
		assert.Contains(t, err.Error(), "Inner2")
	})

	t.Run("invalid - pointer composition error message validation", func(t *testing.T) {
		type ServiceConfig struct {
			Name string `validate:"required"`
		}

		// Mix of pointer and struct fields with required_alone
		input := struct {
			PointerService *ServiceConfig `validate:"omitempty,required_alone"`
			StructService  ServiceConfig  `validate:"required_alone"`
			PlainField     string         `validate:"required_alone"`
		}{
			PointerService: &ServiceConfig{Name: "pointer-service"}, // has value
			StructService:  ServiceConfig{Name: "struct-service"},   // also has value - violation
			PlainField:     "",                                      // empty
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "required_alone")
		assert.Contains(t, err.Error(), "PointerService")
		assert.Contains(t, err.Error(), "StructService")
	})

	t.Run("invalid - embedded struct composition error contains validation details", func(t *testing.T) {
		type Credentials struct {
			Username string `validate:"required_alone"`
			Password string `validate:"required_alone"`
			APIKey   string `validate:"required_alone"`
		}

		type ServiceConfig struct {
			Credentials `validate:"required"` // embedded struct
			Endpoint    string                `validate:"required_alone"`
		}

		// Embedded struct has multiple required_alone fields with values
		input := ServiceConfig{
			Credentials: Credentials{
				Username: "user",     // has value
				Password: "password", // also has value - violation within embedded struct
				APIKey:   "",         // empty
			},
			Endpoint: "", // empty
		}

		err := Validate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "required_alone")
		// Should contain field names from the embedded struct
		errorMsg := err.Error()
		assert.True(t,
			assert.Contains(t, errorMsg, "Username") ||
				assert.Contains(t, errorMsg, "Password"),
			"Error message should contain field names: %s", errorMsg)
	})
}
