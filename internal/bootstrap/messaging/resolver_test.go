package messaging

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	messagingconfig "github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Run("success with default engine", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &redisCfg,
		})
		require.NoError(t, err)

		picker := messagingconfig.Picker{
			Engine: "REDIS",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		_, err = Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.NoError(t, err)

		_, err = Resolve[chainstream.DispatchFailureNotifier](t.Context(), picker)
		assert.NoError(t, err)
	})

	t.Run("success with inline config", func(t *testing.T) {
		rabbitCfg := setupRabbitMQContainer(t)
		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				RabbitMQ: &rabbitCfg,
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{
					RoutingKey: "test-key",
				},
			},
		}

		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.NoError(t, err)

		_, err = Resolve[chainstream.DispatchFailureNotifier](t.Context(), picker)
		assert.NoError(t, err)
	})

	t.Run("unsupported engine in picker", func(t *testing.T) {
		picker := messagingconfig.Picker{
			Engine: "UNKNOWN",
		}
		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("unsupported adapter", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &redisCfg,
		})
		require.NoError(t, err)

		picker := messagingconfig.Picker{
			Engine: "REDIS",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		_, err = Resolve[io.Writer](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("no publisher config", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &redisCfg,
		})
		require.NoError(t, err)

		picker := messagingconfig.Picker{
			Engine: "REDIS",
		}
		_, err = Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("adapter returns wrong type", func(t *testing.T) {
		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: originalFactory.BuildConnection,
			InterfaceAdapters: map[reflect.Type]func(conn, pubCfg any) any{
				typeOf[walletwatch.TransactionNotifier](): func(conn, pubCfg any) any {
					return struct{}{} // not a walletwatch.TransactionNotifier
				},
			},
		}
		t.Cleanup(func() {
			messagingFactories["REDIS"] = originalFactory
		})

		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &redisCfg,
		})
		require.NoError(t, err)

		picker := messagingconfig.Picker{
			Engine: "REDIS",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		_, err = Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("factory build error on inline", func(t *testing.T) {
		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return nil, errors.New("build failed")
			},
		}
		t.Cleanup(func() {
			messagingFactories["REDIS"] = originalFactory
		})

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}
		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("no engine config", func(t *testing.T) {
		picker := messagingconfig.Picker{}
		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("unsupported inline engine", func(t *testing.T) {
		originalFactories := messagingFactories
		messagingFactories = make(map[string]messagingFactory)
		t.Cleanup(func() {
			messagingFactories = originalFactories
		})

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				RabbitMQ: &messagingconfig.RabbitMQConnection{},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{},
			},
		}
		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})

	t.Run("inline with mismatched publisher", func(t *testing.T) {
		rabbitCfg := setupRabbitMQContainer(t)
		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				RabbitMQ: &rabbitCfg,
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{},
			},
		}
		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)
		assert.Error(t, err)
	})
}

func TestAdaptMessaging(t *testing.T) {
	t.Run("unsupported engine", func(t *testing.T) {
		_, err := adaptMessaging[walletwatch.TransactionNotifier](nil, nil, "UNKNOWN")
		assert.Error(t, err)
	})
}
