package logger

import (
	"bytes"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/gabapcia/blockwatch/internal/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetLogger resets the global logger state for testing
func resetLogger() {
	logger = nil
	initOnce = sync.Once{}
}

func TestWithLevel(t *testing.T) {
	t.Run("debug level", func(t *testing.T) {
		cfg := &config{}
		opt := WithLevel("debug")
		opt(cfg)
		assert.Equal(t, "debug", cfg.level)
	})

	t.Run("info level", func(t *testing.T) {
		cfg := &config{}
		opt := WithLevel("info")
		opt(cfg)
		assert.Equal(t, "info", cfg.level)
	})

	t.Run("warn level", func(t *testing.T) {
		cfg := &config{}
		opt := WithLevel("warn")
		opt(cfg)
		assert.Equal(t, "warn", cfg.level)
	})

	t.Run("error level", func(t *testing.T) {
		cfg := &config{}
		opt := WithLevel("error")
		opt(cfg)
		assert.Equal(t, "error", cfg.level)
	})

	t.Run("panic level", func(t *testing.T) {
		cfg := &config{}
		opt := WithLevel("panic")
		opt(cfg)
		assert.Equal(t, "panic", cfg.level)
	})

	t.Run("fatal level", func(t *testing.T) {
		cfg := &config{}
		opt := WithLevel("fatal")
		opt(cfg)
		assert.Equal(t, "fatal", cfg.level)
	})
}

func TestInit(t *testing.T) {
	t.Run("default initialization", func(t *testing.T) {
		resetLogger()
		err := Init()
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with debug level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("debug"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with info level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("info"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with warn level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("warn"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with error level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("error"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with panic level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("panic"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with fatal level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("fatal"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("with invalid level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("invalid"))
		assert.Error(t, err)
	})

	t.Run("with multiple options", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("debug"), WithLevel("info")) // last one wins
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("init only once", func(t *testing.T) {
		resetLogger()

		// First initialization
		err1 := Init(WithLevel("debug"))
		require.NoError(t, err1)
		firstLogger := logger

		// Second initialization should not change the logger
		err2 := Init(WithLevel("error"))
		require.NoError(t, err2)
		assert.Equal(t, firstLogger, logger, "Init() should only initialize once")
	})

	t.Run("with telemetry integration", func(t *testing.T) {
		resetLogger()

		// Initialize telemetry first to test the integration path
		ctx := t.Context()
		shutdownFunc, err := telemetry.Init(ctx, "test-service")
		if err != nil {
			t.Skip("Telemetry initialization failed, skipping telemetry integration test:", err)
		}
		defer func() {
			if shutdownFunc != nil {
				shutdownFunc(ctx)
			}
		}()

		// Now initialize logger - this should cover the telemetry integration branch
		err = Init(WithLevel("info"))
		require.NoError(t, err)
		assert.NotNil(t, logger)

		// Test that logging still works with telemetry integration
		assert.NotPanics(t, func() {
			Info(ctx, "test message with telemetry")
		})
	})
}

func TestSync(t *testing.T) {
	t.Run("sync after init", func(t *testing.T) {
		resetLogger()
		err := Init()
		require.NoError(t, err)

		// Note: Sync may return an error when stdout is redirected in tests
		// The important thing is that it doesn't panic
		Sync()
	})

	t.Run("sync without init panics", func(t *testing.T) {
		resetLogger()

		assert.Panics(t, func() {
			Sync()
		}, "Sync() should panic when logger is not initialized")
	})
}

func TestDebug(t *testing.T) {
	t.Run("debug with message and key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("debug"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test debug message"
		key := "testKey"
		value := "testValue"

		// Test that Debug doesn't panic
		assert.NotPanics(t, func() {
			Debug(ctx, msg, key, value)
		})
	})

	t.Run("debug without key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("debug"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test debug message without context"

		// Test that Debug doesn't panic
		assert.NotPanics(t, func() {
			Debug(ctx, msg)
		})
	})

	t.Run("debug with different log level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("info"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test debug message"

		// Test that Debug doesn't panic even when log level is higher
		assert.NotPanics(t, func() {
			Debug(ctx, msg)
		})
	})

	t.Run("key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("debug"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test debug message with context"

		t.Run("single key-value pair", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1"}

			// Test that Debug doesn't panic with key-value pairs
			assert.NotPanics(t, func() {
				Debug(ctx, msg, keysAndValues...)
			})
		})

		t.Run("multiple key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2", 42, "key3", true}

			// Test that Debug doesn't panic with multiple key-value pairs
			assert.NotPanics(t, func() {
				Debug(ctx, msg, keysAndValues...)
			})
		})

		t.Run("no key-value pairs", func(t *testing.T) {
			keysAndValues := []any{}

			// Test that Debug doesn't panic with no key-value pairs
			assert.NotPanics(t, func() {
				Debug(ctx, msg, keysAndValues...)
			})
		})

		t.Run("odd number of key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2"} // Missing value for key2

			// Test that Debug doesn't panic with odd number of arguments
			assert.NotPanics(t, func() {
				Debug(ctx, msg, keysAndValues...)
			})
		})
	})
}

func TestInfo(t *testing.T) {
	t.Run("info with message and key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("info"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test info message"
		key := "infoKey"
		value := 42

		// Test that Info doesn't panic
		assert.NotPanics(t, func() {
			Info(ctx, msg, key, value)
		})
	})

	t.Run("info without key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("info"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test info message without context"

		// Test that Info doesn't panic
		assert.NotPanics(t, func() {
			Info(ctx, msg)
		})
	})

	t.Run("info with different log level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("warn"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test info message"

		// Test that Info doesn't panic even when log level is higher
		assert.NotPanics(t, func() {
			Info(ctx, msg)
		})
	})

	t.Run("key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("info"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test info message with context"

		t.Run("single key-value pair", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1"}

			// Test that Info doesn't panic with key-value pairs
			assert.NotPanics(t, func() {
				Info(ctx, msg, keysAndValues...)
			})
		})

		t.Run("multiple key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2", 42, "key3", true}

			// Test that Info doesn't panic with multiple key-value pairs
			assert.NotPanics(t, func() {
				Info(ctx, msg, keysAndValues...)
			})
		})

		t.Run("no key-value pairs", func(t *testing.T) {
			keysAndValues := []any{}

			// Test that Info doesn't panic with no key-value pairs
			assert.NotPanics(t, func() {
				Info(ctx, msg, keysAndValues...)
			})
		})

		t.Run("odd number of key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2"} // Missing value for key2

			// Test that Info doesn't panic with odd number of arguments
			assert.NotPanics(t, func() {
				Info(ctx, msg, keysAndValues...)
			})
		})
	})
}

func TestWarn(t *testing.T) {
	t.Run("warn with message", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("warn"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test warn message"

		// Test that Warn doesn't panic
		assert.NotPanics(t, func() {
			Warn(ctx, msg)
		})
	})

	t.Run("warn with key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("warn"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test warn message"
		key := "warnKey"
		value := "warnValue"

		// Test that Warn doesn't panic
		assert.NotPanics(t, func() {
			Warn(ctx, msg, key, value)
		})
	})

	t.Run("warn with different log level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("error"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test warn message"

		// Test that Warn doesn't panic even when log level is higher
		assert.NotPanics(t, func() {
			Warn(ctx, msg)
		})
	})

	t.Run("key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("warn"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test warn message with context"

		t.Run("single key-value pair", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1"}

			// Test that Warn doesn't panic with key-value pairs
			assert.NotPanics(t, func() {
				Warn(ctx, msg, keysAndValues...)
			})
		})

		t.Run("multiple key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2", 42, "key3", true}

			// Test that Warn doesn't panic with multiple key-value pairs
			assert.NotPanics(t, func() {
				Warn(ctx, msg, keysAndValues...)
			})
		})

		t.Run("no key-value pairs", func(t *testing.T) {
			keysAndValues := []any{}

			// Test that Warn doesn't panic with no key-value pairs
			assert.NotPanics(t, func() {
				Warn(ctx, msg, keysAndValues...)
			})
		})

		t.Run("odd number of key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2"} // Missing value for key2

			// Test that Warn doesn't panic with odd number of arguments
			assert.NotPanics(t, func() {
				Warn(ctx, msg, keysAndValues...)
			})
		})
	})
}

func TestError(t *testing.T) {
	t.Run("error with message and key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("error"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test error message"
		errorKey := "error"
		errorValue := "something went wrong"

		// Test that Error doesn't panic
		assert.NotPanics(t, func() {
			Error(ctx, msg, errorKey, errorValue)
		})
	})

	t.Run("error without key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("error"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test error message without context"

		// Test that Error doesn't panic
		assert.NotPanics(t, func() {
			Error(ctx, msg)
		})
	})

	t.Run("error with same log level", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("error"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test error message"

		// Test that Error doesn't panic
		assert.NotPanics(t, func() {
			Error(ctx, msg)
		})
	})

	t.Run("key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("error"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test error message with context"

		t.Run("single key-value pair", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1"}

			// Test that Error doesn't panic with key-value pairs
			assert.NotPanics(t, func() {
				Error(ctx, msg, keysAndValues...)
			})
		})

		t.Run("multiple key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2", 42, "key3", true}

			// Test that Error doesn't panic with multiple key-value pairs
			assert.NotPanics(t, func() {
				Error(ctx, msg, keysAndValues...)
			})
		})

		t.Run("no key-value pairs", func(t *testing.T) {
			keysAndValues := []any{}

			// Test that Error doesn't panic with no key-value pairs
			assert.NotPanics(t, func() {
				Error(ctx, msg, keysAndValues...)
			})
		})

		t.Run("odd number of key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2"} // Missing value for key2

			// Test that Error doesn't panic with odd number of arguments
			assert.NotPanics(t, func() {
				Error(ctx, msg, keysAndValues...)
			})
		})
	})
}

func TestPanic(t *testing.T) {
	t.Run("panic with message", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("panic"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test panic message"

		assert.Panics(t, func() {
			Panic(ctx, msg)
		}, "Panic() should panic")
	})

	t.Run("panic with key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("panic"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test panic message"
		key := "panicKey"
		value := "panicValue"

		assert.Panics(t, func() {
			Panic(ctx, msg, key, value)
		}, "Panic() should panic")
	})

	t.Run("key-value pairs", func(t *testing.T) {
		resetLogger()
		err := Init(WithLevel("panic"))
		require.NoError(t, err)

		ctx := t.Context()
		msg := "test panic message with context"

		t.Run("single key-value pair", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1"}

			// Test that Panic panics with key-value pairs
			assert.Panics(t, func() {
				Panic(ctx, msg, keysAndValues...)
			})
		})

		t.Run("multiple key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2", 42, "key3", true}

			// Test that Panic panics with multiple key-value pairs
			assert.Panics(t, func() {
				Panic(ctx, msg, keysAndValues...)
			})
		})

		t.Run("no key-value pairs", func(t *testing.T) {
			keysAndValues := []any{}

			// Test that Panic panics with no key-value pairs
			assert.Panics(t, func() {
				Panic(ctx, msg, keysAndValues...)
			})
		})

		t.Run("odd number of key-value pairs", func(t *testing.T) {
			keysAndValues := []any{"key1", "value1", "key2"} // Missing value for key2

			// Test that Panic panics with odd number of arguments
			assert.Panics(t, func() {
				Panic(ctx, msg, keysAndValues...)
			})
		})
	})
}

func TestFatal(t *testing.T) {
	t.Run("fatal exits with code 1", func(t *testing.T) {
		// This subprocess will execute the Fatal call.
		if os.Getenv("TEST_FATAL_SUBPROCESS") == "1" {
			// initialize logger (so logger.Fatal is wired up)
			_ = Init(WithLevel("debug"))
			// this will call os.Exit(1)
			Fatal(t.Context(), "fatal error for test")
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

	t.Run("key-value pairs", func(t *testing.T) {
		t.Run("single key-value pair", func(t *testing.T) {
			// This subprocess will execute the Fatal call.
			if os.Getenv("TEST_FATAL_SINGLE_KV_SUBPROCESS") == "1" {
				// initialize logger (so logger.Fatal is wired up)
				_ = Init(WithLevel("debug"))
				keysAndValues := []any{"key1", "value1"}
				// this will call os.Exit(1)
				Fatal(t.Context(), "fatal error for test", keysAndValues...)
				return
			}

			// Build a command that re-runs this test in a subprocess.
			cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
			cmd.Env = append(os.Environ(), "TEST_FATAL_SINGLE_KV_SUBPROCESS=1")

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

		t.Run("multiple key-value pairs", func(t *testing.T) {
			// This subprocess will execute the Fatal call.
			if os.Getenv("TEST_FATAL_MULTIPLE_KV_SUBPROCESS") == "1" {
				// initialize logger (so logger.Fatal is wired up)
				_ = Init(WithLevel("debug"))
				keysAndValues := []any{"key1", "value1", "key2", 42, "key3", true}
				// this will call os.Exit(1)
				Fatal(t.Context(), "fatal error for test", keysAndValues...)
				return
			}

			// Build a command that re-runs this test in a subprocess.
			cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
			cmd.Env = append(os.Environ(), "TEST_FATAL_MULTIPLE_KV_SUBPROCESS=1")

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

		t.Run("no key-value pairs", func(t *testing.T) {
			// This subprocess will execute the Fatal call.
			if os.Getenv("TEST_FATAL_NO_KV_SUBPROCESS") == "1" {
				// initialize logger (so logger.Fatal is wired up)
				_ = Init(WithLevel("debug"))
				keysAndValues := []any{}
				// this will call os.Exit(1)
				Fatal(t.Context(), "fatal error for test", keysAndValues...)
				return
			}

			// Build a command that re-runs this test in a subprocess.
			cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
			cmd.Env = append(os.Environ(), "TEST_FATAL_NO_KV_SUBPROCESS=1")

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

		t.Run("odd number of key-value pairs", func(t *testing.T) {
			// This subprocess will execute the Fatal call.
			if os.Getenv("TEST_FATAL_ODD_KV_SUBPROCESS") == "1" {
				// initialize logger (so logger.Fatal is wired up)
				_ = Init(WithLevel("debug"))
				keysAndValues := []any{"key1", "value1", "key2"} // Missing value for key2
				// this will call os.Exit(1)
				Fatal(t.Context(), "fatal error for test", keysAndValues...)
				return
			}

			// Build a command that re-runs this test in a subprocess.
			cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
			cmd.Env = append(os.Environ(), "TEST_FATAL_ODD_KV_SUBPROCESS=1")

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
	})
}
