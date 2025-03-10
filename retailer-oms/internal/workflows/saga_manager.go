package workflows

import (
	"context"
	"log"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/itimofeev/go-saga"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const sagaContextKey contextKey = "sagaContext"

type ServicesConfig struct {
	SvcAssembly string
	SvcPayment  string
	SvcDelivery string
}

// SagaManager управляет выполнением саги
type SagaManager struct {
	saga           *saga.Saga
	servicesConfig *ServicesConfig
}

type SagaContextData struct {
	LastSpanContext trace.SpanContext
	Order           domain.Order
	ServicesConfig  *ServicesConfig
}

// NewSagaManager создает новый экземпляр SagaManager
func NewSagaManager(cfg *ServicesConfig) *SagaManager {
	sm := &SagaManager{
		saga:           saga.NewSaga("OrderProcessing"),
		servicesConfig: cfg,
	}

	// Функция для безопасного добавления шагов
	addStep := func(step *saga.Step) {
		if err := sm.saga.AddStep(step); err != nil {
			log.Fatalf("Ошибка при добавлении шага %s: %v", step.Name, err)
		}
	}

	// Регистрация шагов саги
	addStep(&saga.Step{
		Name:           "AcceptOrder",
		Func:           sm.wrapAction(AcceptOrder),
		CompensateFunc: sm.wrapAction(CancelOrder),
	})

	addStep(&saga.Step{
		Name:           "AssembleOrder",
		Func:           sm.wrapAction(AssembleOrder),
		CompensateFunc: sm.wrapAction(ReturnToStock),
	})

	addStep(&saga.Step{
		Name:           "PayOrder",
		Func:           sm.wrapAction(PayOrder),
		CompensateFunc: sm.wrapAction(RefundPayment),
	})

	addStep(&saga.Step{
		Name:           "ShipOrder",
		Func:           sm.wrapAction(ShipOrder),
		CompensateFunc: sm.wrapAction(ReturnToWarehouse),
	})

	addStep(&saga.Step{
		Name:           "CompleteOrder",
		Func:           sm.wrapAction(CompleteOrder),
		CompensateFunc: func(ctx context.Context) error { return nil }, // Финальный шаг, без компенсации
	})

	return sm
}

// Execute запускает сагу
func (sm *SagaManager) Execute(ctx context.Context, order domain.Order) error {
	span := trace.SpanFromContext(ctx)
	sagaCtxData := &SagaContextData{LastSpanContext: span.SpanContext(), Order: order, ServicesConfig: sm.servicesConfig}
	// New context from background
	sagaCtx := context.WithValue(context.Background(), sagaContextKey, sagaCtxData)

	coordinator := saga.NewCoordinator(sagaCtx, sagaCtx, sm.saga, saga.New())

	result := coordinator.Play()
	if result.ExecutionError != nil {
		log.Printf("Ошибка выполнения саги: %v", result.ExecutionError)
		return result.ExecutionError
	}
	log.Println("Сага выполнена успешно")
	return nil
}

// wrapAction оборачивает действие в обработку OpenTelemetry
func (sm *SagaManager) wrapAction(action func(context.Context, *SagaContextData) error) func(context.Context) error {
	return func(ctx context.Context) error {
		// Получаем SagaContext
		sagaCtx, _ := ctx.Value(sagaContextKey).(*SagaContextData)
		if sagaCtx == nil {
			sagaCtx = &SagaContextData{}
		}

		return action(ctx, sagaCtx)
	}
}
