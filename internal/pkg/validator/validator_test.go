package validator

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	RequiredField string `validate:"required"`
	NumberField   int    `validate:"gt=10"`
}

func TestMain(m *testing.M) {
	Init()
	m.Run()
}

func TestValidate(t *testing.T) {
	t.Run("successful validation", func(t *testing.T) {
		validStruct := testStruct{
			RequiredField: "some value",
			NumberField:   11,
		}

		err := Validate(validStruct)
		assert.NoError(t, err)
	})

	t.Run("validation fails with one error", func(t *testing.T) {
		invalidStruct := testStruct{
			RequiredField: "some value",
			NumberField:   5,
		}

		err := Validate(invalidStruct)
		require.Error(t, err)

		assert.True(t, errors.Is(err, ErrValidation))

		expectedErr := fmt.Errorf(errStringFormat, "NumberField", 5, "gt")
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("validation fails with multiple errors", func(t *testing.T) {
		invalidStruct := testStruct{
			RequiredField: "",
			NumberField:   5,
		}

		err := Validate(invalidStruct)
		require.Error(t, err)

		assert.True(t, errors.Is(err, ErrValidation))

		expectedErr1 := fmt.Errorf(errStringFormat, "RequiredField", "", "required")
		assert.Contains(t, err.Error(), expectedErr1.Error())

		expectedErr2 := fmt.Errorf(errStringFormat, "NumberField", 5, "gt")
		assert.Contains(t, err.Error(), expectedErr2.Error())
	})

	t.Run("validation on non-struct type", func(t *testing.T) {
		err := Validate("a string")
		assert.Error(t, err)
	})
}

func TestInit(t *testing.T) {
	t.Run("init is idempotent", func(t *testing.T) {
		// Init() is called in TestMain, so we call it again to test idempotency
		assert.NotPanics(t, func() {
			Init()
		})
	})
}
