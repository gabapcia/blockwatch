// Package logger provides a global, Sugared Zap logger with optional
// OpenTelemetry integration. It supports configuring log level via functional
// options, emits JSON logs to stdout, and automatically adds an OTEL bridge
// core when a telemetry provider is available.
package logger

import (
	"context"
	"os"
	"sync"

	"github.com/gabapcia/blockwatch/internal/pkg/telemetry"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// logger is the global SugaredLogger instance. It is initialized once by Init.
	logger *zap.SugaredLogger

	// initOnce ensures the logger is only configured a single time.
	initOnce sync.Once
)

// config holds configuration options for the logger.
type config struct {
	level string // the minimum log level (debug, info, warn, error, panic, fatal)
}

// Option configures the logger before initialization.
type Option func(*config)

// WithLevel sets the minimum log level for the global logger.
// Example levels: "debug", "info", "warn", "error", "panic", "fatal".
func WithLevel(l string) Option {
	return func(c *config) {
		c.level = l
	}
}

// Init configures the global logger. It accepts zero or more Option values to
// customize behavior (e.g. WithLevel). By default, it logs JSON to stdout at
// the "info" level. If an OpenTelemetry LoggerProvider is registered via
// telemetry.LoggerProvider(), this adds an OTEL bridge core to forward logs to
// the telemetry backend. Calling Init multiple times has no effect after the
// first successful initialization.
//
// Returns an error if parsing the log level fails.
func Init(opts ...Option) error {
	cfg := config{level: "info"}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Parse the configured log level.
	level, err := zapcore.ParseLevel(cfg.level)
	if err != nil {
		return err
	}

	// Perform one-time setup.
	initOnce.Do(func() {
		// Base core: JSON encoder writing to stdout.
		cores := []zapcore.Core{
			zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(os.Stdout),
				level,
			),
		}

		// If telemetry is configured, add OTEL bridge core.
		if lp := telemetry.LoggerProvider(); lp != nil {
			cores = append(cores, otelzap.NewCore("", otelzap.WithLoggerProvider(lp)))
		}

		logger = zap.New(zapcore.NewTee(cores...)).Sugar()
	})

	return nil
}

// Sync flushes any buffered log entries. It should be called on application
// shutdown to ensure all logs are written out.
func Sync() error {
	return logger.Sync()
}

// Debug logs a debug-level message with optional key/value context.
func Debug(ctx context.Context, msg string, keysAndValues ...any) {
	logger.Debugw(msg, keysAndValues...)
}

// Info logs an info-level message with optional key/value context.
func Info(ctx context.Context, msg string, keysAndValues ...any) {
	logger.Infow(msg, keysAndValues...)
}

// Warn logs a warn-level message with optional key/value context.
func Warn(ctx context.Context, msg string, keysAndValues ...any) {
	logger.Warnw(msg, keysAndValues...)
}

// Error logs an error-level message with optional key/value context.
func Error(ctx context.Context, msg string, keysAndValues ...any) {
	logger.Errorw(msg, keysAndValues...)
}

// Panic logs a panic-level message (and then panics) with optional key/value context.
func Panic(ctx context.Context, msg string, keysAndValues ...any) {
	logger.Panicw(msg, keysAndValues...)
}

// Fatal logs a fatal-level message (and then exits) with optional key/value context.
func Fatal(ctx context.Context, msg string, keysAndValues ...any) {
	logger.Fatalw(msg, keysAndValues...)
}
