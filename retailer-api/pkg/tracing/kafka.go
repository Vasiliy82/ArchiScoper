package tracing

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// InjectTraceContextToKafka добавляет `traceparent` в заголовки Kafka
func InjectTraceContextToKafka(ctx context.Context) []kgo.RecordHeader {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)

	traceparent, ok := carrier["traceparent"]
	if !ok {
		return []kgo.RecordHeader{} // Если трассировки нет, передаём пустой заголовок
	}

	return []kgo.RecordHeader{
		{Key: "traceparent", Value: []byte(traceparent)},
	}
}

// ExtractTraceContextFromKafka извлекает `traceparent` из заголовков Kafka и создаёт `Link` (Consumer)
func ExtractTraceContextFromKafka(ctx context.Context, headers []kgo.RecordHeader) []trace.Link {
	carrier := propagation.MapCarrier{}

	// Ищем `traceparent` в заголовках Kafka
	for _, h := range headers {
		if h.Key == "traceparent" {
			carrier["traceparent"] = string(h.Value)
			break
		}
	}
	if carrier["traceparent"] == "" {
		return nil
	}

	// Извлекаем контекст трассировки из Kafka-заголовков
	parentCtx := propagation.TraceContext{}.Extract(ctx, carrier)
	parentSpanCtx := trace.SpanContextFromContext(parentCtx)

	// Создаём Link с архитектурными атрибутами
	var links []trace.Link
	if parentSpanCtx.IsValid() {
		links = append(links, trace.Link{
			SpanContext: parentSpanCtx,
			Attributes: []attribute.KeyValue{
				attribute.String("link.type", "async"),     // Kafka - асинхронная связь
				attribute.String("link.protocol", "kafka"), // Указываем, что это Kafka
				attribute.String("link.role", "consumer"),  // Этот сервис - consumer
			},
		})
	}

	return links
}
