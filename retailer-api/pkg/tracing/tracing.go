package tracing

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceNameTemplate = "%s.%s.%s"
)

// Определяем глобальный трейсер
var tracer trace.Tracer

// InitTracer настраивает OpenTelemetry Tracer Provider
func InitTracer(cfg TraceConfig, info AppInfo) func() {
	// Настройка экспортера
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.ExporterURL),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(cfg.Timeout),
	)
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.Fatalf("Ошибка инициализации OTLP экспортера: %v", err)
	}

	// Создание Tracer Provider
	tp := traceSdk.NewTracerProvider(
		traceSdk.WithSampler(traceSdk.ParentBased(traceSdk.TraceIDRatioBased(cfg.SampleRate))),
		traceSdk.WithBatcher(exporter),
		traceSdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(info.ServiceName),
			semconv.ServiceVersion(info.ServiceVersion),
			semconv.DeploymentEnvironment(info.Environment),
			semconv.SourceDomain(info.DomainName),
			semconv.ServiceInstanceID(info.ServiceInstanceID),
		)),
	)

	otel.SetTracerProvider(tp)

	tracer = tp.Tracer(fmt.Sprintf(traceNameTemplate, info.DomainName, info.ServiceName, info.ServiceVersion))

	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Завершающий обработчик
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Ошибка при завершении TracerProvider: %v", err)
		}
	}

}

// StartApplication создает span для бизнес-логики (UseCase)
func StartApplication(ctx context.Context, operation string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, operation, LayerApplication, SubLayerUseCase, opts...)
}

// StartPresentation создает span для входного слоя (HTTP/gRPC)
func StartPresentation(ctx context.Context, operation string, subLayer SubLayer, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, operation, LayerPresentation, subLayer, opts...)
}

// StartInfrastructure создает span для инфраструктурных операций (БД, кэш, брокеры)
func StartInfrastructure(ctx context.Context, operation string, subLayer SubLayer, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, operation, LayerInfrastructure, subLayer, opts...)
}

// StartIntegration создает span для внешних сервисов
func StartIntegration(ctx context.Context, operation string, subLayer SubLayer, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, operation, LayerIntegration, subLayer, opts...)
}

// startSpan - общий метод для создания span с правильными атрибутами
func startSpan(ctx context.Context, operation string, layer Layer, subLayer SubLayer, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Проверяем допустимую комбинацию Layer → SubLayer
	if err := validateLayerSubLayer(layer, subLayer); err != nil {
		panic(fmt.Sprintf("Ошибка трассировки: %v", err)) // Паника здесь уместна, так как ошибка программиста
	}

	// Определяем имя функции
	functionName := getCallerFunctionName()

	// Формируем имя спана
	spanName := generateSpanName(operation, layer, subLayer)

	opts = append(opts,
		trace.WithAttributes(
			attribute.String("layer", string(layer)),
			attribute.String("subLayer", string(subLayer)),
			attribute.String("function.name", functionName),
		),
	)

	// Создаем span
	ctx, span := tracer.Start(ctx, spanName,
		opts...,
	)

	return ctx, span
}
