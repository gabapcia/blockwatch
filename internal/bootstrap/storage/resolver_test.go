package storage

import (
	"context"
	"errors"
	"testing"

	storageconfig "github.com/gabapcia/blockwatch/internal/pkg/config/storage"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	t.Run("by engine name", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), storageconfig.Engines{
			Redis: &redisCfg,
		})
		assert.NoError(t, err)

		type redisInstance interface{}
		_, err = Resolve[redisInstance](t.Context(), storageconfig.Picker{
			Engine: "REDIS",
		})
		assert.NoError(t, err)
	})

	t.Run("by engine name with invalid name", func(t *testing.T) {
		type invalidInstance interface{}
		_, err := Resolve[invalidInstance](t.Context(), storageconfig.Picker{
			Engine: "INVALID",
		})
		assert.Error(t, err)
	})

	t.Run("by engine name with unexpected type", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), storageconfig.Engines{
			Redis: &redisCfg,
		})
		assert.NoError(t, err)

		type someOtherInterface interface {
			DoSomething()
		}
		_, err = Resolve[someOtherInterface](t.Context(), storageconfig.Picker{
			Engine: "REDIS",
		})
		assert.Error(t, err)
	})

	t.Run("with inline config", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)

		type redisInstance interface{}
		_, err := Resolve[redisInstance](t.Context(), storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &redisCfg,
			},
		})
		assert.NoError(t, err)
	})

	t.Run("with inline config and unsupported engine", func(t *testing.T) {
		originalFactories := storageFactories
		storageFactories = make(map[string]storageFactory)
		t.Cleanup(func() {
			storageFactories = originalFactories
		})

		type someInterface interface{}
		_, err := Resolve[someInterface](t.Context(), storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				PostgreSQL: &storageconfig.PostgreSQL{},
			},
		})
		assert.Error(t, err)
	})

	t.Run("with inline config and factory error", func(t *testing.T) {
		originalRedisFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return nil, errors.New("factory failed")
		}
		t.Cleanup(func() {
			storageFactories["REDIS"] = originalRedisFactory
		})

		type redisInstance interface{}
		_, err := Resolve[redisInstance](t.Context(), storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{},
			},
		})
		assert.Error(t, err)
	})

	t.Run("with inline config and unexpected type", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)

		type someOtherInterface interface {
			DoSomething()
		}
		_, err := Resolve[someOtherInterface](t.Context(), storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &redisCfg,
			},
		})
		assert.Error(t, err)
	})

	t.Run("with no valid config", func(t *testing.T) {
		type someInterface interface{}
		_, err := Resolve[someInterface](t.Context(), storageconfig.Picker{})
		assert.Error(t, err)
	})

	t.Run("with inline postgresql config", func(t *testing.T) {
		postgresCfg := setupPostgresContainer(t)

		type postgresInstance interface{}
		_, err := Resolve[postgresInstance](t.Context(), storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				PostgreSQL: &postgresCfg,
			},
		})
		assert.NoError(t, err)
	})
}
