// Package logger provides a global, context-aware Sugared Zap logger.
// It is intended to be initialized once during application bootstrap
// and then used throughout the codebase via context propagation.
//
// The logger automatically enriches log entries with OpenTelemetry trace and span IDs
// when present in the context. It also allows deriving named loggers with structured fields.
package logger

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// key is an unexported type used to store a logger in the context.
type key struct{}

var (
	// ctxKey is the context key used to associate a logger with a context.
	ctxKey = key{}

	// baseLogger is the global SugaredLogger initialized by Init.
	baseLogger *zap.SugaredLogger

	// initBaseLoggerOnce ensures Init runs only once.
	initBaseLoggerOnce sync.Once
)

// Init initializes the global base logger.
//
// It creates a JSON-encoded logger that writes to stdout and sets the specified
// log level. It must be called once before any logging is performed.
//
// Calling Init multiple times has no effect after the first successful call.
//
// Returns an error if the log level string is invalid.
func Init(serviceName, level string) error {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return err
	}

	initBaseLoggerOnce.Do(func() {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			lvl,
		)

		baseLogger = zap.New(core).Named(serviceName).Sugar()
	})

	return nil
}

// deriveFromCtx returns a logger enriched with contextual fields,
// such as trace/span IDs and user-defined key-value pairs.
//
// If the context does not contain a derived logger, it uses the global baseLogger.
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

// Derive returns a new context containing a named logger enriched
// with optional structured fields. Useful for creating scoped loggers
// for components, handlers, or goroutines.
func Derive(ctx context.Context, name string, keysAndValues ...any) context.Context {
	derivedLogger := deriveFromCtx(ctx, keysAndValues...)
	return context.WithValue(ctx, ctxKey, derivedLogger.Named(name))
}

// Sync flushes any buffered logs to the output stream.
//
// It should be called during application shutdown to ensure all log entries
// are flushed.
func Sync() error {
	if baseLogger == nil {
		return nil
	}

	return baseLogger.Sync()
}

// log retrieves a context-aware logger and emits a structured log entry
// at the specified zap log level.
func log(ctx context.Context, lvl zapcore.Level, msg string, keysAndValues ...any) {
	logger, ok := ctx.Value(ctxKey).(*zap.SugaredLogger)
	if !ok {
		logger = deriveFromCtx(ctx)
	}

	logger.Logw(lvl, msg, keysAndValues...)
}

// Debug logs a debug-level message with optional structured fields.
func Debug(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.DebugLevel, msg, keysAndValues...)
}

// Info logs an info-level message with optional structured fields.
func Info(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.InfoLevel, msg, keysAndValues...)
}

// Warn logs a warning-level message with optional structured fields.
func Warn(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.WarnLevel, msg, keysAndValues...)
}

// Error logs an error-level message with optional structured fields.
func Error(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.ErrorLevel, msg, keysAndValues...)
}

// Panic logs a panic-level message with optional structured fields and panics.
func Panic(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.PanicLevel, msg, keysAndValues...)
}

// Fatal logs a fatal-level message with optional structured fields and terminates the process.
func Fatal(ctx context.Context, msg string, keysAndValues ...any) {
	log(ctx, zapcore.FatalLevel, msg, keysAndValues...)
}
