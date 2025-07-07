package logger

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// resetLogger resets the global logger state for testing
func resetLogger() {
	baseLogger = nil
	initBaseLoggerOnce = sync.Once{}
}

func TestInit(t *testing.T) {
	t.Run("successful initialization with valid level", func(t *testing.T) {
		resetLogger()
		err := Init("info")
		require.NoError(t, err)
		assert.NotNil(t, baseLogger)
	})

	t.Run("successful initialization with debug level", func(t *testing.T) {
		resetLogger()
		err := Init("debug")
		require.NoError(t, err)
		assert.NotNil(t, baseLogger)
	})

	t.Run("successful initialization with warn level", func(t *testing.T) {
		resetLogger()
		err := Init("warn")
		require.NoError(t, err)
		assert.NotNil(t, baseLogger)
	})

	t.Run("successful initialization with error level", func(t *testing.T) {
		resetLogger()
		err := Init("error")
		require.NoError(t, err)
		assert.NotNil(t, baseLogger)
	})

	t.Run("error with invalid level", func(t *testing.T) {
		resetLogger()
		err := Init("invalid")
		assert.Error(t, err)
		assert.Nil(t, baseLogger)
	})

	t.Run("init only once", func(t *testing.T) {
		resetLogger()

		// First initialization
		err1 := Init("debug")
		require.NoError(t, err1)
		firstLogger := baseLogger

		// Second initialization should not change the logger
		err2 := Init("error")
		require.NoError(t, err2)
		assert.Equal(t, firstLogger, baseLogger, "Init() should only initialize once")
	})
}

func TestDeriveFromCtx(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("derive from context without logger", func(t *testing.T) {
		ctx := t.Context()
		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})

	t.Run("derive from context with existing logger", func(t *testing.T) {
		ctx := t.Context()
		existingLogger := baseLogger.With("existing", "logger")
		ctx = context.WithValue(ctx, ctxKey, existingLogger)

		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})

	t.Run("derive with trace context", func(t *testing.T) {
		// Create a tracer and span
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(t.Context(), "test-span")
		defer span.End()

		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})

	t.Run("derive with no additional keys", func(t *testing.T) {
		ctx := t.Context()
		logger := deriveFromCtx(ctx)
		assert.NotNil(t, logger)
	})

	t.Run("derive with multiple key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		logger := deriveFromCtx(ctx, "key1", "value1", "key2", 42, "key3", true)
		assert.NotNil(t, logger)
	})
}

func TestDerive(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("derive context with key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		derivedCtx := Derive(ctx, "key", "value")

		// Check that the derived context contains a logger
		logger, ok := derivedCtx.Value(ctxKey).(*zap.SugaredLogger)
		assert.True(t, ok)
		assert.NotNil(t, logger)
	})

	t.Run("derive context with no key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		derivedCtx := Derive(ctx)

		// Check that the derived context contains a logger
		logger, ok := derivedCtx.Value(ctxKey).(*zap.SugaredLogger)
		assert.True(t, ok)
		assert.NotNil(t, logger)
	})

	t.Run("derive context with multiple key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		derivedCtx := Derive(ctx, "key1", "value1", "key2", 42)

		// Check that the derived context contains a logger
		logger, ok := derivedCtx.Value(ctxKey).(*zap.SugaredLogger)
		assert.True(t, ok)
		assert.NotNil(t, logger)
	})
}

func TestSync(t *testing.T) {
	t.Run("sync after init", func(t *testing.T) {
		resetLogger()
		err := Init("info")
		require.NoError(t, err)

		// Sync should not panic and may return an error (which is fine for stdout)
		assert.NotPanics(t, func() {
			Sync()
		})
	})

	t.Run("sync without init panics", func(t *testing.T) {
		resetLogger()

		assert.Panics(t, func() {
			Sync()
		}, "Sync() should panic when logger is not initialized")
	})
}

func TestLog(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("log with context containing logger", func(t *testing.T) {
		ctx := Derive(t.Context(), "test", "value")

		assert.NotPanics(t, func() {
			log(ctx, zapcore.InfoLevel, "test message", "key", "value")
		})
	})

	t.Run("log with context without logger", func(t *testing.T) {
		ctx := t.Context()

		assert.NotPanics(t, func() {
			log(ctx, zapcore.InfoLevel, "test message", "key", "value")
		})
	})

	t.Run("log with different levels", func(t *testing.T) {
		ctx := t.Context()

		levels := []zapcore.Level{
			zapcore.DebugLevel,
			zapcore.InfoLevel,
			zapcore.WarnLevel,
			zapcore.ErrorLevel,
		}

		for _, level := range levels {
			assert.NotPanics(t, func() {
				log(ctx, level, "test message", "key", "value")
			})
		}
	})
}

func TestDebug(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("debug with message and key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Debug(ctx, "debug message", "key", "value")
		})
	})

	t.Run("debug without key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Debug(ctx, "debug message")
		})
	})

	t.Run("debug with derived context", func(t *testing.T) {
		ctx := Derive(t.Context(), "context", "derived")
		assert.NotPanics(t, func() {
			Debug(ctx, "debug message", "key", "value")
		})
	})
}

func TestInfo(t *testing.T) {
	resetLogger()
	err := Init("info")
	require.NoError(t, err)

	t.Run("info with message and key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Info(ctx, "info message", "key", "value")
		})
	})

	t.Run("info without key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Info(ctx, "info message")
		})
	})

	t.Run("info with derived context", func(t *testing.T) {
		ctx := Derive(t.Context(), "context", "derived")
		assert.NotPanics(t, func() {
			Info(ctx, "info message", "key", "value")
		})
	})
}

func TestWarn(t *testing.T) {
	resetLogger()
	err := Init("warn")
	require.NoError(t, err)

	t.Run("warn with message and key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Warn(ctx, "warn message", "key", "value")
		})
	})

	t.Run("warn without key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Warn(ctx, "warn message")
		})
	})

	t.Run("warn with derived context", func(t *testing.T) {
		ctx := Derive(t.Context(), "context", "derived")
		assert.NotPanics(t, func() {
			Warn(ctx, "warn message", "key", "value")
		})
	})
}

func TestError(t *testing.T) {
	resetLogger()
	err := Init("error")
	require.NoError(t, err)

	t.Run("error with message and key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Error(ctx, "error message", "key", "value")
		})
	})

	t.Run("error without key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Error(ctx, "error message")
		})
	})

	t.Run("error with derived context", func(t *testing.T) {
		ctx := Derive(t.Context(), "context", "derived")
		assert.NotPanics(t, func() {
			Error(ctx, "error message", "key", "value")
		})
	})
}

func TestPanic(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("panic with message", func(t *testing.T) {
		ctx := t.Context()
		assert.Panics(t, func() {
			Panic(ctx, "panic message")
		}, "Panic() should panic")
	})

	t.Run("panic with key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.Panics(t, func() {
			Panic(ctx, "panic message", "key", "value")
		}, "Panic() should panic")
	})

	t.Run("panic with derived context", func(t *testing.T) {
		ctx := Derive(t.Context(), "context", "derived")
		assert.Panics(t, func() {
			Panic(ctx, "panic message", "key", "value")
		}, "Panic() should panic")
	})
}

func TestFatal(t *testing.T) {
	t.Run("fatal exits with code 1", func(t *testing.T) {
		// This subprocess will execute the Fatal call.
		if os.Getenv("TEST_FATAL_SUBPROCESS") == "1" {
			// initialize logger
			_ = Init("debug")
			// this will call os.Exit(1)
			Fatal(context.Background(), "fatal error for test")
			return
		}

		// Build a command that re-runs this test in a subprocess.
		cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
		cmd.Env = append(os.Environ(), "TEST_FATAL_SUBPROCESS=1")

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		exitErr, ok := err.(*exec.ExitError)
		assert.True(t, ok, "the subprocess should exit with a non-zero status")
		assert.Equal(t, 1, exitErr.ExitCode(), "logger.Fatal should terminate with exit code 1")

		// Assert that the log message appears on stdout (logger writes to stdout):
		assert.Contains(t, stdout.String(), `"level":"fatal"`)
	})

	t.Run("fatal with key-value pairs", func(t *testing.T) {
		// This subprocess will execute the Fatal call.
		if os.Getenv("TEST_FATAL_KV_SUBPROCESS") == "1" {
			// initialize logger
			_ = Init("debug")
			// this will call os.Exit(1)
			Fatal(context.Background(), "fatal error for test", "key", "value")
			return
		}

		// Build a command that re-runs this test in a subprocess.
		cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
		cmd.Env = append(os.Environ(), "TEST_FATAL_KV_SUBPROCESS=1")

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		exitErr, ok := err.(*exec.ExitError)
		assert.True(t, ok, "the subprocess should exit with a non-zero status")
		assert.Equal(t, 1, exitErr.ExitCode(), "logger.Fatal should terminate with exit code 1")

		// Assert that the log message appears on stdout:
		assert.Contains(t, stdout.String(), `"level":"fatal"`)
	})

	t.Run("fatal with derived context", func(t *testing.T) {
		// This subprocess will execute the Fatal call.
		if os.Getenv("TEST_FATAL_DERIVED_SUBPROCESS") == "1" {
			// initialize logger
			_ = Init("debug")
			ctx := Derive(context.Background(), "context", "derived")
			// this will call os.Exit(1)
			Fatal(ctx, "fatal error for test", "key", "value")
			return
		}

		// Build a command that re-runs this test in a subprocess.
		cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
		cmd.Env = append(os.Environ(), "TEST_FATAL_DERIVED_SUBPROCESS=1")

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		exitErr, ok := err.(*exec.ExitError)
		assert.True(t, ok, "the subprocess should exit with a non-zero status")
		assert.Equal(t, 1, exitErr.ExitCode(), "logger.Fatal should terminate with exit code 1")

		// Assert that the log message appears on stdout:
		assert.Contains(t, stdout.String(), `"level":"fatal"`)
	})
}

func TestTraceIntegration(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("logging with trace context", func(t *testing.T) {
		// Create a tracer and span
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(t.Context(), "test-span")
		defer span.End()

		// Test that logging with trace context doesn't panic
		assert.NotPanics(t, func() {
			Info(ctx, "test message with trace", "key", "value")
		})
	})

	t.Run("derive context with trace", func(t *testing.T) {
		// Create a tracer and span
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(t.Context(), "test-span")
		defer span.End()

		// Derive context with trace
		derivedCtx := Derive(ctx, "derived", "value")

		// Test that logging with derived trace context doesn't panic
		assert.NotPanics(t, func() {
			Info(derivedCtx, "test message with derived trace", "key", "value")
		})
	})

	t.Run("derive with context containing valid span context", func(t *testing.T) {
		// Create a span context with valid trace and span IDs
		traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
		spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		})

		// Create context with the span context
		ctx := trace.ContextWithSpanContext(t.Context(), spanContext)

		// Test deriveFromCtx with valid span context
		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})

	t.Run("derive with context containing span context with only trace ID", func(t *testing.T) {
		// Create a span context with only trace ID
		traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
		})

		// Create context with the span context
		ctx := trace.ContextWithSpanContext(t.Context(), spanContext)

		// Test deriveFromCtx with trace ID only
		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})

	t.Run("derive with context containing span context with only span ID", func(t *testing.T) {
		// Create a span context with only span ID
		spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			SpanID: spanID,
		})

		// Create context with the span context
		ctx := trace.ContextWithSpanContext(t.Context(), spanContext)

		// Test deriveFromCtx with span ID only
		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})

	t.Run("derive with empty span context", func(t *testing.T) {
		// Create an empty span context
		spanContext := trace.SpanContext{}

		// Create context with the empty span context
		ctx := trace.ContextWithSpanContext(t.Context(), spanContext)

		// Test deriveFromCtx with empty span context
		logger := deriveFromCtx(ctx, "key", "value")
		assert.NotNil(t, logger)
	})
}

func TestEdgeCases(t *testing.T) {
	resetLogger()
	err := Init("debug")
	require.NoError(t, err)

	t.Run("nil context values", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Info(ctx, "test message", "key", nil)
		})
	})

	t.Run("empty message", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Info(ctx, "")
		})
	})

	t.Run("odd number of key-value pairs", func(t *testing.T) {
		ctx := t.Context()
		assert.NotPanics(t, func() {
			Info(ctx, "test message", "key1", "value1", "key2")
		})
	})

	t.Run("complex value types", func(t *testing.T) {
		ctx := t.Context()
		complexValue := map[string]interface{}{
			"nested": map[string]string{"key": "value"},
			"array":  []int{1, 2, 3},
		}
		assert.NotPanics(t, func() {
			Info(ctx, "test message", "complex", complexValue)
		})
	})
}
