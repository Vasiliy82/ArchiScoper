package tracing

import (
	"context"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
)

// Основная обертка для создания спана
func runWithTrace(ctx context.Context, operation string, layer Layer, subLayer SubLayer, fn func(ctx context.Context) error, attrs ...attribute.KeyValue) error {
	// Проверяем слои
	if err := validateLayerSubLayer(layer, subLayer); err != nil {
		return fmt.Errorf("tracing error: %w", err)
	}

	// Определяем имя функции
	functionName := getCallerFunctionName()

	// Формируем имя спана
	spanName := generateSpanName(operation, layer, subLayer)

	// Создаем спан
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()
	spanAttrs := append(attrs, attribute.String("layer", string(layer)),
		attribute.String("subLayer", string(subLayer)),
		attribute.String("function.name", functionName),
	)

	// Добавляем атрибуты
	span.SetAttributes(
		spanAttrs...,
	)

	// Выполняем переданную функцию
	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Получение имени вызывающей функции
func getCallerFunctionName() string {
	pc, _, _, ok := runtime.Caller(2) // 2 уровня вверх
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

// Формирование имени спана по шаблону
func generateSpanName(operation string, layer Layer, subLayer SubLayer) string {
	// Базовое имя
	spanName := operation

	// Добавляем слой, если это необходимо
	if layer == LayerApplication {
		spanName = fmt.Sprintf("Business %s", operation)
	} else if layer == LayerInfrastructure && subLayer == SubLayerDatabase {
		spanName = fmt.Sprintf("SQL %s", operation)
	} else if layer == LayerIntegration && subLayer == SubLayerThirdParty {
		spanName = fmt.Sprintf("External API %s", operation)
	} else if layer == LayerInfrastructure && subLayer == SubLayerBroker {
		spanName = fmt.Sprintf("Kafka %s", operation)
	}

	return spanName
}
