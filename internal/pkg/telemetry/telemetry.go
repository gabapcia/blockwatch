// Package telemetry provides helpers to initialize OpenTelemetry logging,
// metrics, and tracing with OTLP exporters over gRPC. It creates a unified
// Resource for the service, registers global providers, and exposes a
// ShutdownFunc to cleanly flush and stop all telemetry pipelines.
package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

var loggerProvider *sdklog.LoggerProvider

// initLoggerProvider sets up an OTLP gRPC LoggerProvider using a
// batched processor and the given Resource. It stores the provider
// globally and returns it for direct use or shutdown.
func initLoggerProvider(ctx context.Context, res *sdkresource.Resource) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	loggerProvider = lp
	return lp, nil
}

// LoggerProvider returns the globally configured LoggerProvider.
// It will be non-nil after Init has been called successfully.
func LoggerProvider() *sdklog.LoggerProvider {
	return loggerProvider
}

// initMeterProvider sets up an OTLP gRPC MeterProvider using a
// periodic reader and the given Resource. It also registers the
// provider as the global MeterProvider.
func initMeterProvider(ctx context.Context, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)
	return mp, nil
}

// initTracerProvider sets up an OTLP gRPC TracerProvider using a
// batched exporter and the given Resource. It also registers the
// provider as the global TracerProvider.
func initTracerProvider(ctx context.Context, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// newResource constructs an OpenTelemetry Resource by merging the default
// system resource with a ServiceName attribute for the given service.
func newResource(serviceName string) (*sdkresource.Resource, error) {
	return sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
}

// ShutdownFunc defines a callback to flush and stop all telemetry providers.
// Call this function at application shutdown to ensure all telemetry is sent.
type ShutdownFunc func(ctx context.Context) error

// Init configures OpenTelemetry for logs, metrics, and traces using OTLP gRPC.
// It takes a context and the logical serviceName, and returns:
//   - a ShutdownFunc to call during application teardown,
//   - or an error if any step fails.
//
// The returned ShutdownFunc will cleanly shutdown metrics, tracer, and logger.
func Init(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	res, err := newResource(serviceName)
	if err != nil {
		return nil, err
	}

	mp, err := initMeterProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	tp, err := initTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	lp, err := initLoggerProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context) error {
		errs := []error{
			mp.Shutdown(ctx),
			tp.Shutdown(ctx),
			lp.Shutdown(ctx),
		}
		return errors.Join(errs...)
	}, nil
}
