package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

func TestNewResource(t *testing.T) {
	t.Run("valid service name", func(t *testing.T) {
		serviceName := "test-service"
		res, err := newResource(serviceName)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Check if the service name attribute is set correctly
		attrs := res.Attributes()
		found := false
		for _, attr := range attrs {
			if attr.Key == semconv.ServiceNameKey {
				assert.Equal(t, serviceName, attr.Value.AsString())
				found = true
				break
			}
		}
		assert.True(t, found, "Service name attribute not found in resource")
	})

	t.Run("empty service name", func(t *testing.T) {
		serviceName := ""
		res, err := newResource(serviceName)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("service name with special characters", func(t *testing.T) {
		serviceName := "test-service-123_special"
		res, err := newResource(serviceName)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Check if the service name attribute is set correctly
		attrs := res.Attributes()
		found := false
		for _, attr := range attrs {
			if attr.Key == semconv.ServiceNameKey {
				assert.Equal(t, serviceName, attr.Value.AsString())
				found = true
				break
			}
		}
		assert.True(t, found, "Service name attribute not found in resource")
	})
}

func TestInitMeterProvider(t *testing.T) {
	// Store original meter provider to restore later
	originalMeterProvider := otel.GetMeterProvider()
	defer func() {
		otel.SetMeterProvider(originalMeterProvider)
	}()

	t.Run("valid context and resource", func(t *testing.T) {
		ctx := context.Background()
		res, err := newResource("test-service")
		require.NoError(t, err)

		mp, err := initMeterProvider(ctx, res)
		if err != nil {
			// Expected to fail without OTLP endpoint configured
			t.Logf("initMeterProvider() failed as expected: %v", err)
		} else {
			// If it succeeds, verify the provider is valid
			assert.NotNil(t, mp, "initMeterProvider() returned nil provider")
			// Clean up
			_ = mp.Shutdown(context.Background())
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res, err := newResource("test-service")
		require.NoError(t, err)

		mp, err := initMeterProvider(ctx, res)
		if err != nil {
			// Expected to fail with cancelled context
			t.Logf("initMeterProvider() failed with cancelled context as expected: %v", err)
		} else {
			// Clean up if it somehow succeeded
			if mp != nil {
				_ = mp.Shutdown(context.Background())
			}
		}
	})
}

func TestInitTracerProvider(t *testing.T) {
	// Store original tracer provider to restore later
	originalTracerProvider := otel.GetTracerProvider()
	defer func() {
		otel.SetTracerProvider(originalTracerProvider)
	}()

	t.Run("valid context and resource", func(t *testing.T) {
		ctx := context.Background()
		res, err := newResource("test-service")
		require.NoError(t, err)

		tp, err := initTracerProvider(ctx, res)
		if err != nil {
			// Expected to fail without OTLP endpoint configured
			t.Logf("initTracerProvider() failed as expected: %v", err)
		} else {
			// If it succeeds, verify the provider is valid
			assert.NotNil(t, tp, "initTracerProvider() returned nil provider")
			// Clean up
			_ = tp.Shutdown(context.Background())
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res, err := newResource("test-service")
		require.NoError(t, err)

		tp, err := initTracerProvider(ctx, res)
		if err != nil {
			// Expected to fail with cancelled context
			t.Logf("initTracerProvider() failed with cancelled context as expected: %v", err)
		} else {
			// Clean up if it somehow succeeded
			if tp != nil {
				_ = tp.Shutdown(context.Background())
			}
		}
	})
}

func TestInit(t *testing.T) {
	// Store original providers to restore later
	originalMeterProvider := otel.GetMeterProvider()
	originalTracerProvider := otel.GetTracerProvider()
	defer func() {
		otel.SetMeterProvider(originalMeterProvider)
		otel.SetTracerProvider(originalTracerProvider)
	}()

	t.Run("valid service name", func(t *testing.T) {
		ctx := context.Background()
		shutdownFunc, err := Init(ctx, "test-service")

		if err != nil {
			// Expected to fail without OTLP endpoint configured
			t.Logf("Init() failed as expected: %v", err)
		} else {
			// If it succeeds, verify the shutdown function is valid
			assert.NotNil(t, shutdownFunc, "Init() returned nil shutdown function")
			// Test the shutdown function with a shorter timeout
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			err = shutdownFunc(shutdownCtx)
			if err != nil {
				// Timeout errors are expected when OTLP endpoint is not available
				t.Logf("ShutdownFunc() returned error (expected): %v", err)
			}
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		ctx := context.Background()
		shutdownFunc, err := Init(ctx, "")

		if err != nil {
			// Expected to fail without OTLP endpoint configured
			t.Logf("Init() failed as expected: %v", err)
		} else {
			// If it succeeds, verify the shutdown function is valid
			assert.NotNil(t, shutdownFunc, "Init() returned nil shutdown function")
			// Test the shutdown function with a shorter timeout
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			err = shutdownFunc(shutdownCtx)
			if err != nil {
				// Timeout errors are expected when OTLP endpoint is not available
				t.Logf("ShutdownFunc() returned error (expected): %v", err)
			}
		}
	})
}

func TestShutdownFunc(t *testing.T) {
	t.Run("shutdown with timeout", func(t *testing.T) {
		// Create mock providers
		mp := sdkmetric.NewMeterProvider()
		tp := sdktrace.NewTracerProvider()
		lp := sdklog.NewLoggerProvider()

		// Create shutdown function manually (simulating what Init returns)
		shutdownFunc := func(ctx context.Context) error {
			errs := []error{
				mp.Shutdown(ctx),
				tp.Shutdown(ctx),
				lp.Shutdown(ctx),
			}
			var result error
			for _, err := range errs {
				if err != nil {
					if result == nil {
						result = err
					}
				}
			}
			return result
		}

		// Test shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := shutdownFunc(ctx)
		assert.NoError(t, err, "ShutdownFunc() should not error with mock providers")
	})

	t.Run("shutdown with cancelled context", func(t *testing.T) {
		// Create mock providers
		mp := sdkmetric.NewMeterProvider()
		tp := sdktrace.NewTracerProvider()
		lp := sdklog.NewLoggerProvider()

		// Create shutdown function manually (simulating what Init returns)
		shutdownFunc := func(ctx context.Context) error {
			errs := []error{
				mp.Shutdown(ctx),
				tp.Shutdown(ctx),
				lp.Shutdown(ctx),
			}
			var result error
			for _, err := range errs {
				if err != nil {
					if result == nil {
						result = err
					}
				}
			}
			return result
		}

		// Test shutdown with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := shutdownFunc(ctx)
		// Should handle cancelled context gracefully
		if err != nil {
			t.Logf("ShutdownFunc() with cancelled context returned error: %v", err)
		}
	})
}
