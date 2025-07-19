package messaging

import (
	"reflect"
	"testing"

	validatorMocks "github.com/gabapcia/blockwatch/internal/pkg/validator/mocks"

	"github.com/stretchr/testify/mock"
)

func TestValidatePickerStruct(t *testing.T) {
	t.Run("valid Redis engine with Redis publisher", func(t *testing.T) {
		picker := Picker{
			Engine: EngineRedis,
			MessagePublisher: MessagePublisher{
				Redis: &RedisPublisher{Stream: "test-stream"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("valid RabbitMQ engine with RabbitMQ publisher", func(t *testing.T) {
		picker := Picker{
			Engine: EngineRabbitMQ,
			MessagePublisher: MessagePublisher{
				RabbitMQ: &RabbitMQPublisher{RoutingKey: "test.key"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("valid Redis inline config with Redis publisher", func(t *testing.T) {
		picker := Picker{
			InlineConfig: InlineConfig{
				Redis: &RedisConnection{Address: "localhost:6379"},
			},
			MessagePublisher: MessagePublisher{
				Redis: &RedisPublisher{Stream: "test-stream"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("valid RabbitMQ inline config with RabbitMQ publisher", func(t *testing.T) {
		picker := Picker{
			InlineConfig: InlineConfig{
				RabbitMQ: &RabbitMQConnection{URI: "amqp://localhost"},
			},
			MessagePublisher: MessagePublisher{
				RabbitMQ: &RabbitMQPublisher{RoutingKey: "test.key"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("Redis engine with missing Redis publisher", func(t *testing.T) {
		picker := Picker{
			Engine: EngineRedis,
			MessagePublisher: MessagePublisher{
				RabbitMQ: &RabbitMQPublisher{RoutingKey: "test.key"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))
		mockSL.EXPECT().ReportError(
			mock.Anything, // Use mock.Anything for reflect.Value
			"MessagePublisher.Redis",
			"Redis",
			"required",
			"",
		)

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("RabbitMQ engine with missing RabbitMQ publisher", func(t *testing.T) {
		picker := Picker{
			Engine: EngineRabbitMQ,
			MessagePublisher: MessagePublisher{
				Redis: &RedisPublisher{Stream: "test-stream"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"MessagePublisher.RabbitMQ",
			"RabbitMQ",
			"required",
			"",
		)

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("Redis inline config with missing Redis publisher", func(t *testing.T) {
		picker := Picker{
			InlineConfig: InlineConfig{
				Redis: &RedisConnection{Address: "localhost:6379"},
			},
			MessagePublisher: MessagePublisher{
				RabbitMQ: &RabbitMQPublisher{RoutingKey: "test.key"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"MessagePublisher.Redis",
			"Redis",
			"required",
			"",
		)

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("RabbitMQ inline config with missing RabbitMQ publisher", func(t *testing.T) {
		picker := Picker{
			InlineConfig: InlineConfig{
				RabbitMQ: &RabbitMQConnection{URI: "amqp://localhost"},
			},
			MessagePublisher: MessagePublisher{
				Redis: &RedisPublisher{Stream: "test-stream"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"MessagePublisher.RabbitMQ",
			"RabbitMQ",
			"required",
			"",
		)

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("no engine provided", func(t *testing.T) {
		picker := Picker{
			MessagePublisher: MessagePublisher{
				Redis: &RedisPublisher{Stream: "test-stream"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"InlineConfig",
			"InlineConfig",
			"required_engine",
			"",
		)

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("nil pointer", func(t *testing.T) {
		var picker *Picker = nil

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("unsupported type", func(t *testing.T) {
		unsupported := "not a picker"

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(unsupported))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("pointer to valid picker", func(t *testing.T) {
		picker := &Picker{
			Engine: EngineRedis,
			MessagePublisher: MessagePublisher{
				Redis: &RedisPublisher{Stream: "test-stream"},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(picker))

		validatePickerStruct(mockSL)
		mockSL.AssertExpectations(t)
	})
}
