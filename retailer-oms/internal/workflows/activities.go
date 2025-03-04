package workflows

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Вероятность фейла для каждой операции (0.01 = 1%)
const fpZero = 0.0
const fpLow = 0.05

// Имитация выполнения действия с возможностью фейла
func executeWithChance(ctx context.Context, stepName string, order domain.Order, failProbability float64) error {
	_, span := tracing.StartApplication(ctx, stepName)
	defer span.End()

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
func AcceptOrder(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "AcceptOrder", order, fpLow)
}
func AssembleOrder(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "AssembleOrder", order, fpLow)
}
func PayOrder(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "PayOrder", order, fpLow)
}
func ShipOrder(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "ShipOrder", order, fpLow)
}
func CompleteOrder(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "CompleteOrder", order, fpZero)
}

// Компенсирующие операции
func CancelOrder(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "CancelOrder", order, fpZero)
}
func ReturnToStock(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "ReturnToStock", order, fpZero)
}
func RefundPayment(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "RefundPayment", order, fpZero)
}
func ReturnToWarehouse(ctx context.Context, order domain.Order) error {
	return executeWithChance(ctx, "ReturnToWarehouse", order, fpZero)
}
