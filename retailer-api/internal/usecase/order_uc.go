package usecase

import (
	"context"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/infrastructure"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type OrderUseCase struct {
	orderRepo *infrastructure.OrderRepository
}

func NewOrderUseCase(orderRepo *infrastructure.OrderRepository) *OrderUseCase {
	return &OrderUseCase{orderRepo: orderRepo}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, payload map[string]interface{}) (string, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "CreateOrder")
	defer span.End()

	order := domain.Order{
		ID:      uuid.New().String(),
		Payload: payload,
	}

	err := uc.orderRepo.PublishOrder(ctx, order)
	if err != nil {
		return "", err
	}

	return order.ID, nil
}
