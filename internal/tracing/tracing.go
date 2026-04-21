// Package tracing wires OpenTelemetry tracing for the Shield REST server.
//
// Env vars consumed (all OTel-standard; defaults target the local Alloy stack):
//
//	OTEL_EXPORTER_OTLP_ENDPOINT       default http://localhost:4318
//	OTEL_EXPORTER_OTLP_HEADERS        default Authorization=...
//	OTEL_SERVICE_NAME                 default "shield"
//	OTEL_SDK_ENABLED=true             opt-in switch — tracing is off by default
//
// The otlptracehttp exporter reads these env vars natively; we only set
// defaults when the caller hasn't overridden them.
package tracing

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Init configures the global tracer provider and returns a shutdown function
// that flushes pending spans. Callers should defer the returned shutdown to
// ensure spans reach the collector before process exit.
//
// Tracing is off by default. When OTEL_SDK_ENABLED is not "true", Init is a
// no-op and returns a shutdown that does nothing — so callers can always
// defer it without checking.
func Init(ctx context.Context) (shutdown func(context.Context) error, err error) {
	if os.Getenv("OTEL_SDK_ENABLED") != "true" {
		return func(context.Context) error { return nil }, nil
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_HEADERS") == "" {
		_ = os.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "")
	}
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "shield"
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	// Only service.name — skip resource.Default() to avoid emitting
	// telemetry.sdk.*, process.*, host.* etc. on every span's Resource.
	res := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		return errors.Join(tp.ForceFlush(ctx), tp.Shutdown(ctx))
	}, nil
}
