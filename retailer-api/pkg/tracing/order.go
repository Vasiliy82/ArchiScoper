package tracing

import (
	"fmt"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"go.opentelemetry.io/otel/attribute"
)

// OrderAttributes формирует список атрибутов для OpenTelemetry
func OrderAttributes(order domain.Order) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	}

	// Добавляем payload, если он есть
	for key, value := range order.Payload {
		attrs = append(attrs, attribute.String("order.payload."+key, toString(value)))
	}

	return attrs
}

// toString конвертирует значение в строку
func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return "unknown"
	}
}
