package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/domain"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

type OrderRepository struct {
	writer *kafka.Writer
}

func NewOrderRepository(broker string) *OrderRepository {
	return &OrderRepository{
		writer: &kafka.Writer{
			Addr:  kafka.TCP(broker),
			Topic: "orders",
		},
	}
}

func (r *OrderRepository) PublishOrder(ctx context.Context, order domain.Order) error {
	tracer := otel.Tracer("repository")
	ctx, span := tracer.Start(ctx, "PublishOrder")
	defer span.End()

	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(order.ID),
		Value: data,
	}

	return r.writer.WriteMessages(ctx, message)
}
