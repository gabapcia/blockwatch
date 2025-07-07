// Package logger provides a global, context-aware Sugared Zap logger.
// It is designed to be initialized once at application startup and then used
// throughout the application. The logger can be derived with additional
// context, and it automatically includes OpenTelemetry trace and span IDs in
// the logs if they are present in the context.
package logger

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// key is the type used for the context key. It is an empty struct because it
// is only used as a key and does not carry any value.
type key struct{}

var (
	// ctxKey is the key used to store the logger in the context.
	ctxKey = key{}

	// baseLogger is the global SugaredLogger instance. It is initialized once by Init.
	baseLogger *zap.SugaredLogger

	// initOnce ensures the logger is only configured a single time.
	initBaseLoggerOnce sync.Once
)

// Init configures the global logger. It logs JSON to stdout at the specified
// level. Calling Init multiple times has no effect after the first successful
// initialization.
//
// It returns an error if parsing the log level fails.
func Init(level string) error {
	// Parse the configured log level.
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return err
	}

	// Perform one-time setup.
	initBaseLoggerOnce.Do(func() {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			lvl,
		)

		baseLogger = zap.New(core).Sugar()
	})

	return nil
}

// deriveFromCtx derives a logger from the context and adds the provided
// keys and values to it. If the context does not contain a logger, it uses the
// base logger. It also adds the trace and span IDs to the logger if they are
// present in the context.
func deriveFromCtx(ctx context.Context, keysAndValues ...any) *zap.SugaredLogger {
	logger, ok := ctx.Value(ctxKey).(*zap.SugaredLogger)
	if !ok {
		logger = baseLogger
	}

	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.HasSpanID() {
		keysAndValues = append(keysAndValues, "span_id", spanContext.SpanID())
	}

	if spanContext.HasTraceID() {
		keysAndValues = append(keysAndValues, "trace_id", spanContext.TraceID())
	}

	return logger.With(keysAndValues...)
}

// Derive returns a new context with a logger that has the provided keys and
// values.
func Derive(ctx context.Context, keysAndValues ...any) context.Context {
	derivedLogger := deriveFromCtx(ctx, keysAndValues...)
	return context.WithValue(ctx, ctxKey, derivedLogger)
}

// Sync flushes any buffered log entries. It should be called on application
// shutdown to ensure all logs are written out.
func Sync() error {
	return baseLogger.Sync()
}

// log retrieves the logger from the context, or derives a new one if it is not
// present, and logs a message at the specified level with the provided
// key/value pairs.
func log(ctx context.Context, lvl zapcore.Level, msg string, keysAndValues ...any) {
	logger, ok := ctx.Value(ctxKey).(*zap.SugaredLogger)
	if !ok {
		logger = deriveFromCtx(ctx)
	}

	logger.Logw(lvl, msg, keysAndValues...)
}

// Debug logs a debug-level message with optional key/value context.
func Debug(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.DebugLevel, msg, keysAndValues...)
}

// Info logs an info-level message with optional key/value context.
func Info(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.InfoLevel, msg, keysAndValues...)
}

// Warn logs a warn-level message with optional key/value context.
func Warn(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.WarnLevel, msg, keysAndValues...)
}

// Error logs an error-level message with optional key/value context.
func Error(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.ErrorLevel, msg, keysAndValues...)
}

// Panic logs a panic-level message (and then panics) with optional key/value context.
func Panic(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.PanicLevel, msg, keysAndValues...)
}

// Fatal logs a fatal-level message (and then exits) with optional key/value context.
func Fatal(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.FatalLevel, msg, keysAndValues...)
}
