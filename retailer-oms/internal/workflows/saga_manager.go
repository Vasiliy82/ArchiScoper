package workflows

import (
	"context"
	"log"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/itimofeev/go-saga"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const orderContextKey contextKey = "order"

// SagaManager управляет выполнением саги
type SagaManager struct {
	saga *saga.Saga
}

// NewSagaManager создает новый экземпляр SagaManager
func NewSagaManager() *SagaManager {
	sm := &SagaManager{
		saga: saga.NewSaga("OrderProcessing"),
	}

	// Регистрация шагов саги
	sm.saga.AddStep(&saga.Step{
		Name:           "AcceptOrder",
		Func:           sm.wrapAction(AcceptOrder),
		CompensateFunc: sm.wrapAction(CancelOrder),
	})

	sm.saga.AddStep(&saga.Step{
		Name:           "AssembleOrder",
		Func:           sm.wrapAction(AssembleOrder),
		CompensateFunc: sm.wrapAction(ReturnToStock),
	})

	sm.saga.AddStep(&saga.Step{
		Name:           "PayOrder",
		Func:           sm.wrapAction(PayOrder),
		CompensateFunc: sm.wrapAction(RefundPayment),
	})

	sm.saga.AddStep(&saga.Step{
		Name:           "ShipOrder",
		Func:           sm.wrapAction(ShipOrder),
		CompensateFunc: sm.wrapAction(ReturnToWarehouse),
	})

	sm.saga.AddStep(&saga.Step{
		Name:           "CompleteOrder",
		Func:           sm.wrapAction(CompleteOrder),
		CompensateFunc: func(ctx context.Context, order domain.Order) {}, // Финальный шаг, без компенсации
	})

	return sm
}

// Execute запускает сагу
func (sm *SagaManager) Execute(ctx context.Context, order domain.Order) error {
	ctx = setOrderToContext(ctx, order) // Сохраняем order в контексте
	coordinator := saga.NewCoordinator(ctx, ctx, sm.saga, saga.New())

	result := coordinator.Play()
	if result.ExecutionError != nil {
		log.Printf("Ошибка выполнения саги: %v", result.ExecutionError)
		return result.ExecutionError
	}
	log.Println("Сага выполнена успешно")
	return nil
}

// wrapAction оборачивает действие в обработку OpenTelemetry
func (sm *SagaManager) wrapAction(action func(context.Context, domain.Order) error) func(context.Context) error {
	return func(ctx context.Context) error {

		order, ok := getOrderFromContext(ctx)
		if !ok {
			log.Println("Ошибка: заказ не найден в контексте")
			return nil
		}
		ctx, span := tracing.StartApplication(ctx, "SagaStep",
			trace.WithAttributes(attribute.KeyValue{Key: "order", Value: attribute.StringValue(order.ID)}),
		)
		defer span.End()

		return action(ctx, order)
	}
}

// setOrderToContext сохраняет заказ в контекст
func setOrderToContext(ctx context.Context, order domain.Order) context.Context {
	return context.WithValue(ctx, orderContextKey, order)
}

// getOrderFromContext извлекает заказ из контекста
func getOrderFromContext(ctx context.Context) (domain.Order, bool) {
	order, ok := ctx.Value(orderContextKey).(domain.Order)
	return order, ok
}
