package workflows

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Вероятность фейла для каждой операции (0.01 = 1%)
const fpZero = 0.0
const fpLow = 0.05

// Имитация выполнения действия с возможностью фейла
func executeWithChance(ctx context.Context, sagaCtx *SagaContextData, stepName string, failProbability float64) error {

	order := sagaCtx.Order

	// Создаём Link на предыдущий шаг (если есть)
	var links []trace.Link
	if sagaCtx.LastSpanContext.IsValid() {
		links = append(links, trace.Link{
			SpanContext: sagaCtx.LastSpanContext,
			Attributes: []attribute.KeyValue{
				attribute.String("link.type", "saga"),
			},
		})
	}

	_, span := tracing.StartApplication(ctx, stepName,
		trace.WithAttributes(
			attribute.KeyValue{Key: "order",
				Value: attribute.StringValue(order.ID),
			}),
		trace.WithLinks(links...),
	)
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	time.Sleep(time.Millisecond * time.Duration(rand.Intn(500))) // Имитация работы
	if rand.Float64() < failProbability {
		log.Output(2, "Ошибка выполнения шага: "+stepName)
		err := fmt.Errorf("ошибка выполнения шага: %s", stepName)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	log.Printf("Шаг %s выполнен успешно для заказа %s", stepName, order.ID)
	span.SetAttributes(attribute.String("order.id", order.ID))
	return nil
}

// Шаги основной логики
func AcceptOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "AcceptOrder", fpLow)
}
func AssembleOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "AssembleOrder", fpLow)
}
func PayOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "PayOrder", fpLow)
}
func ShipOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "ShipOrder", fpLow)
}
func CompleteOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "CompleteOrder", fpZero)
}

// Компенсирующие операции
func CancelOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "CancelOrder", fpZero)
}
func ReturnToStock(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "ReturnToStock", fpZero)
}
func RefundPayment(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "RefundPayment", fpZero)
}
func ReturnToWarehouse(ctx context.Context, sagaCtx *SagaContextData) error {
	return executeWithChance(ctx, sagaCtx, "ReturnToWarehouse", fpZero)
}
