package messaging

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/infra/messaging/rabbitmq"
	"github.com/gabapcia/blockwatch/internal/infra/messaging/redis"
	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// messagingFactory defines the constructor and interface adapter registry for a messaging engine.
//
// Each entry provides:
//   - BuildConnection: a function that creates the messaging client based on engine-specific config.
//   - InterfaceAdapters: a map from target interface types to functions that convert the connection
//     to the expected interface using the publisher configuration.
type messagingFactory struct {
	BuildConnection   func(ctx context.Context, config any) (any, error)
	InterfaceAdapters map[reflect.Type]func(conn, pubCfg any) any
}

// messagingFactories maps supported messaging engines to their factory definitions.
//
// Keys must be uppercase engine identifiers (e.g., "REDIS", "RABBITMQ").
//
// To support a new engine:
//  1. Define the connection struct and publisher struct in config/messaging.
//  2. Implement the corresponding client with adapter methods.
//  3. Add a new entry here using BuildConnection and InterfaceAdapters.
//
// Example:
//
//	"KAFKA": {
//		BuildConnection: func(ctx context.Context, cfg any) (any, error) {
//			kafkaCfg := cfg.(messaging.KafkaConnection)
//			return kafka.NewClient(ctx, kafkaCfg.Brokers)
//		},
//		InterfaceAdapters: map[reflect.Type]func(conn, pubCfg any) any{
//			typeOf[events.Dispatcher](): func(conn, pubCfg any) any {
//				cfg := pubCfg.(messaging.KafkaPublisher)
//				return conn.(*kafka.Client).AsDispatcher(cfg.Topic)
//			},
//		},
//	}
var messagingFactories = map[string]messagingFactory{
	"REDIS": {
		BuildConnection: func(ctx context.Context, cfg any) (any, error) {
			redisCfg := cfg.(messaging.RedisConnection)
			return redis.New(ctx, redisCfg.Address, redisCfg.Username, redisCfg.Password, redisCfg.DB)
		},
		InterfaceAdapters: map[reflect.Type]func(conn, pubCfg any) any{
			typeOf[walletwatch.TransactionNotifier](): func(conn, pubCfg any) any {
				cfg := pubCfg.(messaging.RedisPublisher)
				return conn.(*redis.Client).AsWalletwatchTransactionNotifier(cfg.Stream)
			},
			typeOf[chainstream.DispatchFailureNotifier](): func(conn, pubCfg any) any {
				cfg := pubCfg.(messaging.RedisPublisher)
				return conn.(*redis.Client).AsChainstreamDispatchFailureNotifier(cfg.Stream)
			},
		},
	},

	"RABBITMQ": {
		BuildConnection: func(ctx context.Context, cfg any) (any, error) {
			rabbitCfg := cfg.(messaging.RabbitMQConnection)
			return rabbitmq.New(ctx, rabbitCfg.URI)
		},
		InterfaceAdapters: map[reflect.Type]func(conn, pubCfg any) any{
			typeOf[walletwatch.TransactionNotifier](): func(conn, pubCfg any) any {
				cfg := pubCfg.(messaging.RabbitMQPublisher)
				return conn.(*rabbitmq.Client).AsWalletwatchTransactionNotifier(cfg.Exchange, cfg.RoutingKey)
			},
			typeOf[chainstream.DispatchFailureNotifier](): func(conn, pubCfg any) any {
				cfg := pubCfg.(messaging.RabbitMQPublisher)
				return conn.(*rabbitmq.Client).AsChainstreamDispatchFailureNotifier(cfg.Exchange, cfg.RoutingKey)
			},
		},
	},
}

// adaptMessaging converts a messaging connection to the requested interface T.
//
// It uses the InterfaceAdapters map of the given engine to find the corresponding adapter.
//
// Parameters:
//   - conn: messaging connection instance (e.g., *redis.Client, *rabbitmq.Client).
//   - pubCfg: engine-specific publisher configuration (e.g., RedisPublisher).
//   - engineName: name of the messaging engine.
//
// Returns:
//   - Adapted instance of type T.
//   - Error if no adapter exists or the type cast fails.
func adaptMessaging[T any](conn, pubCfg any, engineName string) (T, error) {
	var (
		zero       T
		targetType = typeOf[T]()
	)

	factory, ok := messagingFactories[engineName]
	if !ok {
		return zero, fmt.Errorf("no factory registered for engine %q", engineName)
	}

	adapterFn, ok := factory.InterfaceAdapters[targetType]
	if !ok {
		return zero, fmt.Errorf("no adapter registered for type %s in engine %q", targetType.String(), engineName)
	}

	instance := adapterFn(conn, pubCfg)
	typed, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("adapter for engine %q returned wrong type: expected %s", engineName, targetType.String())
	}

	return typed, nil
}

// extractPublisherConfig returns the appropriate publisher config for the specified engine.
//
// It reflects over the MessagePublisher struct and matches the field by name (case-insensitive).
//
// Parameters:
//   - engineName: name of the messaging engine (must match a struct field).
//   - publishers: MessagePublisher struct containing backend-specific configurations.
//
// Returns:
//   - The matched publisher configuration (e.g., RedisPublisher, RabbitMQPublisher).
//   - An error if no matching configuration is found.
func extractPublisherConfig(engineName string, publishers messaging.MessagePublisher) (any, error) {
	pubVal := reflect.ValueOf(publishers)
	pubType := reflect.TypeOf(publishers)

	for i := 0; i < pubVal.NumField(); i++ {
		field := pubVal.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() &&
			strings.EqualFold(pubType.Field(i).Name, engineName) {
			return field.Elem().Interface(), nil
		}
	}

	return nil, fmt.Errorf("no publisher configuration found for engine %q", engineName)
}

// Resolve selects and returns a messaging instance adapted to the desired interface.
//
// It supports two selection mechanisms:
//   - If Picker.Engine is set, it uses a default instance from the global map.
//   - If InlineConfig is provided, it dynamically creates a new instance using the appropriate factory.
//
// After the connection is resolved, it is adapted to the requested interface T using the
// matching function from the InterfaceAdapters map.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - picker: configuration containing either Engine (default) or InlineConfig.
//
// Returns:
//   - The resolved messaging instance implementing interface T.
//   - An error if resolution fails, the adapter is not registered, or casting fails.
func Resolve[T any](ctx context.Context, picker messaging.Picker) (T, error) {
	var (
		zero       T
		engineName = strings.ToUpper(picker.Engine)
	)

	if engineName != "" {
		conn, found := defaults[engineName]
		if !found {
			return zero, fmt.Errorf("no default messaging instance found for engine %q", engineName)
		}

		pubCfg, err := extractPublisherConfig(engineName, picker.MessagePublisher)
		if err != nil {
			return zero, err
		}

		return adaptMessaging[T](conn, pubCfg, engineName)
	}

	inlineVal := reflect.ValueOf(picker.InlineConfig)
	inlineType := reflect.TypeOf(picker.InlineConfig)

	for i := 0; i < inlineVal.NumField(); i++ {
		field := inlineVal.Field(i)
		if field.Kind() != reflect.Ptr || field.IsNil() {
			continue
		}

		engineName := strings.ToUpper(inlineType.Field(i).Name)
		factory, ok := messagingFactories[engineName]
		if !ok {
			return zero, fmt.Errorf("no messaging factory registered for engine %q", engineName)
		}

		conn, err := factory.BuildConnection(ctx, field.Elem().Interface())
		if err != nil {
			return zero, fmt.Errorf("failed to create inline messaging instance for engine %q: %w", engineName, err)
		}

		if closer, ok := conn.(io.Closer); ok {
			openedConnections = append(openedConnections, closer)
		}

		pubCfg, err := extractPublisherConfig(engineName, picker.MessagePublisher)
		if err != nil {
			return zero, err
		}

		return adaptMessaging[T](conn, pubCfg, engineName)
	}

	return zero, fmt.Errorf("no valid messaging engine configuration provided")
}
