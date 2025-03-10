package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// httpPost делает HTTP-запрос
func httpPost(ctx context.Context, serviceURL string, requestData interface{}) error {
	// Сериализуем тело запроса
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Выполняем HTTP-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания HTTP-запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Добавляем заголовки трассировки
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса в %s: %w", serviceURL, err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ошибка ответа %d от %s", resp.StatusCode, serviceURL)
	}

	return nil
}

// linksFromSagaContext создает trace.Link для связи с предыдущим шагом
func linksFromSagaContext(sagaCtx *SagaContextData) []trace.Link {
	if sagaCtx.LastSpanContext.IsValid() {
		return []trace.Link{
			{
				SpanContext: sagaCtx.LastSpanContext,
				Attributes: []attribute.KeyValue{
					attribute.String("link.type", "saga"), // Kafka - асинхронная связь
				},
			},
		}
	}
	return nil
}

// **Функции активностей**

// Принятие заказа (локальная логика)
func AcceptOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	_, span := tracing.StartApplication(ctx, "AcceptOrder", trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	// Сохраняем ссылку на последний шаг
	sagaCtx.LastSpanContext = span.SpanContext()

	log.Printf("Заказ %s принят", sagaCtx.Order.ID)
	span.SetStatus(codes.Ok, "успешно")
	return nil
}

// Сборка заказа (вызов удаленного сервиса)
func AssembleOrder(ctx context.Context, sagaCtx *SagaContextData) error {

	ctx, span := tracing.StartIntegration(ctx, "AssembleOrder", tracing.SubLayerHTTP, trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	serviceURL := fmt.Sprintf("http://%s/assembly", sagaCtx.ServicesConfig.SvcAssembly)
	err := httpPost(ctx, serviceURL, sagaCtx.Order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "успешно")
	log.Printf("Шаг AssembleOrder успешно выполнен для заказа %s", sagaCtx.Order.ID)
	return nil
}

// Оплата заказа (вызов удаленного сервиса)
func PayOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	ctx, span := tracing.StartIntegration(ctx, "PayOrder", tracing.SubLayerHTTP, trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	serviceURL := fmt.Sprintf("http://%s/payment", sagaCtx.ServicesConfig.SvcPayment)
	err := httpPost(ctx, serviceURL, sagaCtx.Order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "успешно")
	log.Printf("Шаг PayOrder успешно выполнен для заказа %s", sagaCtx.Order.ID)
	return nil
}

// Доставка заказа (вызов удаленного сервиса)
func ShipOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	ctx, span := tracing.StartIntegration(ctx, "ShipOrder", tracing.SubLayerHTTP, trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	serviceURL := fmt.Sprintf("http://%s/delivery", sagaCtx.ServicesConfig.SvcDelivery)
	err := httpPost(ctx, serviceURL, sagaCtx.Order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "успешно")
	log.Printf("Шаг ShipOrder успешно выполнен для заказа %s", sagaCtx.Order.ID)
	return nil
}

// Завершение заказа (локальная логика)
func CompleteOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	_, span := tracing.StartApplication(ctx, "CompleteOrder", trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	log.Printf("Заказ %s успешно завершен", sagaCtx.Order.ID)
	span.SetStatus(codes.Ok, "успешно")
	return nil
}

// **Функции активностей**

// Отмена заказа (локальная логика)
func CancelOrder(ctx context.Context, sagaCtx *SagaContextData) error {
	_, span := tracing.StartApplication(ctx, "CancelOrder", trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	log.Printf("Заказ %s отменен", sagaCtx.Order.ID)
	span.SetStatus(codes.Ok, "успешно")
	return nil
}

// Возврат товара на склад (вызов Assembly Service)
func ReturnToStock(ctx context.Context, sagaCtx *SagaContextData) error {
	ctx, span := tracing.StartIntegration(ctx, "ReturnToStock", tracing.SubLayerHTTP, trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	serviceURL := fmt.Sprintf("http://%s/cancel-assembly", sagaCtx.ServicesConfig.SvcAssembly)
	err := httpPost(ctx, serviceURL, sagaCtx.Order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "успешно")
	log.Printf("Товары заказа %s возвращены на склад", sagaCtx.Order.ID)
	return nil
}

// Возврат денег клиенту (вызов Refund Service)
func RefundPayment(ctx context.Context, sagaCtx *SagaContextData) error {
	ctx, span := tracing.StartIntegration(ctx, "RefundPayment", tracing.SubLayerHTTP, trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	serviceURL := fmt.Sprintf("http://%s/cancel-payment", sagaCtx.ServicesConfig.SvcPayment)
	err := httpPost(ctx, serviceURL, sagaCtx.Order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "успешно")
	log.Printf("Деньги за заказ %s возвращены клиенту", sagaCtx.Order.ID)
	return nil
}

// Возврат заказа на склад (вызов Refund Service)
func ReturnToWarehouse(ctx context.Context, sagaCtx *SagaContextData) error {
	ctx, span := tracing.StartIntegration(ctx, "ReturnToWarehouse", tracing.SubLayerHTTP, trace.WithLinks(linksFromSagaContext(sagaCtx)...))
	defer span.End()

	sagaCtx.LastSpanContext = span.SpanContext()

	serviceURL := fmt.Sprintf("http://%s/cancel-delivery", sagaCtx.ServicesConfig.SvcDelivery)
	err := httpPost(ctx, serviceURL, sagaCtx.Order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "успешно")
	log.Printf("Заказ %s возвращен на склад", sagaCtx.Order.ID)
	return nil
}
