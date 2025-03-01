package usecase

import (
	"context"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/infrastructure"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// OrderUseCase содержит бизнес-логику работы с заказами
type OrderUseCase struct {
	orderRepo *infrastructure.OrderRepository
}

// NewOrderUseCase создает экземпляр OrderUseCase
func NewOrderUseCase(orderRepo *infrastructure.OrderRepository) *OrderUseCase {
	return &OrderUseCase{orderRepo: orderRepo}
}

// CreateOrder создает новый заказ и публикует его в Kafka
func (uc *OrderUseCase) CreateOrder(ctx context.Context, payload map[string]interface{}) (string, error) {
	ctx, span := tracing.StartApplication(ctx, "CreateOrder")
	defer span.End()

	order := domain.Order{
		ID:      uuid.New().String(),
		Payload: payload,
	}

	span.SetAttributes(
		attribute.String("order.id", order.ID),
		attribute.Int("payload.size", len(payload)),
	)

	err := uc.orderRepo.PublishOrder(ctx, order)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	return order.ID, nil
}
