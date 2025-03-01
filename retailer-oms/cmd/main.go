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
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
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
	temporalTaskQueue := os.Getenv("TEMPORAL_ORDER_QUEUE")
	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")

	// Подключение к Temporal
	temporalClient, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Ошибка подключения к Temporal: %v", err)
	}
	defer temporalClient.Close()

	// Запуск Kafka-консьюмера
	kafkaConsumer := infrastructure.NewKafkaConsumer(kafkaBrokers, kafkaTopic, temporalClient)
	defer kafkaConsumer.Close()

	// Запуск Temporal Worker
	wkr := worker.New(temporalClient, temporalTaskQueue, worker.Options{})
	wkr.RegisterWorkflow(workflows.OrderWorkflow)
	wkr.RegisterActivity(workflows.AcceptOrder)
	wkr.RegisterActivity(workflows.AssembleOrder)
	wkr.RegisterActivity(workflows.PayOrder)
	wkr.RegisterActivity(workflows.ShipOrder)
	wkr.RegisterActivity(workflows.CompleteOrder)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go kafkaConsumer.StartListening(ctx)

	// Запуск воркера в отдельной горутине
	go func() {
		log.Println("Запуск Temporal Worker...")
		if err := wkr.Run(worker.InterruptCh()); err != nil {
			log.Fatalf("Ошибка при запуске Temporal Worker: %v", err)
		}
	}()

	// Ожидание сигнала завершения работы
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Завершаем работу...")
	cancel()

}
