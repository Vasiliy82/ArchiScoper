package tracing

import (
	"context"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracer настраивает OpenTelemetry Tracer Provider
func InitTracer(cfg Config) func() {
	// Настройка экспортера
	client := otlptracehttp.NewClient(otlptracehttp.WithEndpoint(cfg.ExporterURL), otlptracehttp.WithInsecure())
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.Fatalf("Ошибка инициализации OTLP экспортера: %v", err)
	}

	// Создание Tracer Provider
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRate))),
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(cfg.Environment),
		)),
	)

	otel.SetTracerProvider(tp)

	// Завершающий обработчик
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Ошибка при завершении TracerProvider: %v", err)
		}
	}
}

// WrapHTTPHandler оборачивает HTTP-хендлер в OpenTelemetry middleware
func WrapHTTPHandler(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, "http-server")
}
