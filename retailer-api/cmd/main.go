package main

import (
	"os"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/handler"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/infrastructure"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/usecase"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/gin-gonic/gin"
)

func main() {

	cfg := tracing.Config{
		ServiceName: "retailer-api",
		ExporterURL: os.Getenv("OTEL_EXPORTER_URL"),
		SampleRate:  1.0,
		Environment: "dev",
	}

	tracing.InitTracer(cfg)

	kafkaBroker := os.Getenv("KAFKA_BROKER")
	orderRepo := infrastructure.NewOrderRepository(kafkaBroker)
	orderUC := usecase.NewOrderUseCase(orderRepo)
	orderHandler := handler.NewOrderHandler(orderUC)

	router := gin.Default()
	router.POST("/orders", orderHandler.CreateOrder)
	router.Run(":8081")
}
