package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-mockservice/internal/config"
	"github.com/Vasiliy82/ArchiScoper/retailer-mockservice/internal/server"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
)

func main() {
	// Загружаем конфиг
	cfg := config.LoadConfig()

	// Инициализация OpenTelemetry трейсов
	tracerCleanup := tracing.InitTracer(
		tracing.TraceConfig{ExporterURL: cfg.ExporterURL, SampleRate: 1.0, Timeout: 5 * time.Second},
		tracing.AppInfo{
			Environment:       "production",
			DomainName:        cfg.DomainName,
			ServiceName:       cfg.ServiceName,
			ServiceVersion:    cfg.ServiceVersion,
			ServiceInstanceID: cfg.ServiceInstanceID,
		},
	)
	defer tracerCleanup()

	// Создаем HTTP сервер
	srv := server.New(cfg)

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Service %s.%s started at %s", cfg.DomainName, cfg.ServiceName, cfg.ListenAddress)
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Обрабатываем сигналы завершения работы
	gracefulShutdown(srv)
}

// gracefulShutdown корректно завершает работу сервиса
func gracefulShutdown(srv *server.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
