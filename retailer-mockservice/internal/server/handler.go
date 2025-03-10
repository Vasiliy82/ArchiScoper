package server

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-mockservice/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
)

func createHandler(cfg config.EndpointConfig) http.HandlerFunc {
	log.Printf("Created endpoint %s %s", cfg.Method, cfg.Name)
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем заголовки трассировки из входящего запроса
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		// Стартуем span для входного HTTP-запроса
		_, span := tracing.StartPresentation(ctx, "HandleRequest", tracing.SubLayerHTTP)
		defer span.End()

		time.Sleep(time.Millisecond * time.Duration(cfg.DurationMin+rand.Intn(int(cfg.DurationRnd)))) // Имитация работы
		// Генерируем ответ с учетом вероятностей ошибок
		prob := rand.Float64()
		switch {
		case prob < cfg.Error400Prob:
			errMsg := "bad request"
			span.SetStatus(codes.Error, errMsg)
			span.SetAttributes(attribute.String("error", errMsg))
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		case prob < cfg.Error400Prob+cfg.Error500Prob:
			errMsg := "internal server error"
			span.SetStatus(codes.Error, errMsg)
			span.SetAttributes(attribute.String("error", errMsg))
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Возвращаем успешный ответ
		span.SetStatus(codes.Ok, "success")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Записываем ответ и проверяем ошибку
		if _, err := w.Write([]byte(cfg.ResponseBody)); err != nil {
			errMsg := "failed to write response"
			span.SetStatus(codes.Error, errMsg)
			span.SetAttributes(
				attribute.String("error", errMsg),
				attribute.String("write_error", err.Error()),
			)
		}
	}
}
