package workflows

import (
	"context"
	"log"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// AcceptOrder принимает заказ
func AcceptOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_, span := tracing.StartApplication(ctx, "AcceptOrder")
	defer span.End()

	log.Printf("Order %s accepted", order.ID)
	order.Status = domain.StatusAccepted

	span.SetAttributes(attribute.String("order.id", order.ID))
	return order, nil
}

// AssembleOrder собирает заказ
func AssembleOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_, span := tracing.StartApplication(ctx, "AssembleOrder")
	defer span.End()

	log.Printf("Order %s assembled", order.ID)
	order.Status = domain.StatusAssembled

	span.SetAttributes(attribute.String("order.id", order.ID))
	return order, nil
}

// PayOrder оплачивает заказ
func PayOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_, span := tracing.StartApplication(ctx, "PayOrder")
	defer span.End()

	log.Printf("Order %s paid", order.ID)
	order.Status = domain.StatusPaid

	span.SetAttributes(attribute.String("order.id", order.ID))
	return order, nil
}

// ShipOrder отправляет заказ
func ShipOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_, span := tracing.StartApplication(ctx, "ShipOrder")
	defer span.End()

	log.Printf("Order %s shipped", order.ID)
	order.Status = domain.StatusShipped

	span.SetAttributes(attribute.String("order.id", order.ID))
	return order, nil
}

// CompleteOrder завершает заказ
func CompleteOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_, span := tracing.StartApplication(ctx, "CompleteOrder")
	defer span.End()

	log.Printf("Order %s completed", order.ID)
	order.Status = domain.StatusCompleted

	span.SetAttributes(attribute.String("order.id", order.ID))
	return order, nil
}
