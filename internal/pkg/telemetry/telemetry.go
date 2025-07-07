// Package telemetry provides helpers to initialize OpenTelemetry logging,
// metrics, and tracing with OTLP exporters over gRPC. It creates a unified
// Resource for the service, registers global providers, and exposes a
// ShutdownFunc to cleanly flush and stop all telemetry pipelines.
package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

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

// Init configures OpenTelemetry for metrics and traces using OTLP over gRPC.
// It initializes the necessary providers for telemetry data collection and export.
//
// Parameters:
//   - ctx: A context.Context for managing the initialization process.
//   - serviceName: A string representing the logical name of the service, used to
//     identify telemetry data in the observability backend.
//
// Returns:
//   - ShutdownFunc: A function to be called during application shutdown to ensure
//     all telemetry data is flushed and providers are stopped gracefully.
//   - error: An error if any part of the initialization process fails.
//
// The returned ShutdownFunc will handle the clean shutdown of metrics and tracer
// providers, ensuring no data is lost during application termination. Note that
// logging setup might be handled separately or integrated based on the application's
// configuration.
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

	return func(ctx context.Context) error {
		errs := []error{
			mp.Shutdown(ctx),
			tp.Shutdown(ctx),
		}
		return errors.Join(errs...)
	}, nil
}
