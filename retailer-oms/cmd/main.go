package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/Vasiliy82/ArchiScoper/retailer-oms/internal/infrastructure"
	"github.com/Vasiliy82/ArchiScoper/retailer-oms/internal/workflows"
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

	// Инициализация трейсинга
	shutdown := tracing.InitTracer(cfg, appInfo)
	defer shutdown()

	kafkaTopic := os.Getenv("KAFKA_ORDERS_TOPIC")
	kafkaBrokers := strings.Split(os.Getenv("KAFKA_BROKER"), ",")

	// Инициализация Saga Manager
	sagaManager := workflows.NewSagaManager()

	// Запуск Kafka-консьюмера
	kafkaConsumer := infrastructure.NewKafkaConsumer(kafkaBrokers, kafkaTopic, sagaManager, 500)
	defer kafkaConsumer.Close()

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go kafkaConsumer.StartListening(ctx)

	// Ожидание сигнала завершения работы
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Завершаем работу...")
	cancel()

}
