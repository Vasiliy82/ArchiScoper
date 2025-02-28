package main

import (
	"os"
	"strings"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/handler"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/infrastructure"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/usecase"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/gin-gonic/gin"
)

func main() {

	cfg := tracing.TraceConfig{
		ExporterURL: os.Getenv("OTEL_EXPORTER_URL"),
		SampleRate:  1.0,
	}
	appInfo := tracing.AppInfo{
		Environment:       os.Getenv("APP_ENV"),             // Окружение (dev, staging, production)
		DomainName:        os.Getenv("APP_DOMAIN"),          // Домен приложения / системы
		ServiceName:       os.Getenv("APP_SERVICE_NAME"),    // Название сервиса
		ServiceVersion:    os.Getenv("APP_SERVICE_VERSION"), // Версия сервиса
		ServiceInstanceID: os.Getenv("APP_INSTANCE_ID"),     // Уникальный ID экземпляра сервиса

	}

	tracing.InitTracer(cfg, appInfo)

	kafkaBrokers := strings.Split(os.Getenv("KAFKA_BROKER"), ",")
	topic := os.Getenv("KAFKA_ORDERS_TOPIC")
	orderRepo := infrastructure.NewOrderRepository(kafkaBrokers, topic)
	orderUC := usecase.NewOrderUseCase(orderRepo)
	orderHandler := handler.NewOrderHandler(orderUC)

	router := gin.Default()
	router.POST("/orders", orderHandler.CreateOrder)
	router.Run(":8081")
}
