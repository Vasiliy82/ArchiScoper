package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/Vasiliy82/ArchiScoper/retailer-oms/internal/workflows"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/trace"
)

// KafkaConsumer отвечает за получение заказов из Kafka
type KafkaConsumer struct {
	client      *kgo.Client
	topic       string
	sagaManager *workflows.SagaManager
	workerCount int
}

// NewKafkaConsumer создает нового Kafka-консьюмера
func NewKafkaConsumer(brokers []string, topic string, sagaManager *workflows.SagaManager, workerCount int) *KafkaConsumer {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("order-management"),
		kgo.ConsumeTopics(topic),
		kgo.BlockRebalanceOnPoll(),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatalf("Ошибка инициализации Kafka-консьюмера: %v", err)
	}

	return &KafkaConsumer{
		client:      client,
		topic:       topic,
		sagaManager: sagaManager,
		workerCount: workerCount,
	}
}

// StartListening запускает обработку сообщений
func (kc *KafkaConsumer) StartListening(ctx context.Context) {
	log.Println("Topic listening started")

	// Канал для передачи сообщений воркерам
	recordsChan := make(chan *kgo.Record, kc.workerCount*2) // Небольшой буфер для баланса нагрузки
	var wg sync.WaitGroup

	// Запускаем N воркеров
	for i := 0; i < kc.workerCount; i++ {
		wg.Add(1)
		go kc.worker(ctx, recordsChan, &wg)
	}

	// Читаем из Kafka и отправляем в канал
	for {
		select {
		case <-ctx.Done():
			close(recordsChan)
			wg.Wait()
			return
		default:
			fetches := kc.client.PollFetches(ctx)
			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				select {
				case recordsChan <- record: // Отправляем запись в канал
				case <-ctx.Done():
					close(recordsChan)
					wg.Wait()
					return
				}
			}
		}
	}
}

// worker обрабатывает сообщения из Kafka
func (kc *KafkaConsumer) worker(ctx context.Context, recordsChan <-chan *kgo.Record, wg *sync.WaitGroup) {
	defer wg.Done()

	for record := range recordsChan {
		func(ctx context.Context) {
			links := tracing.ExtractTraceContextFromKafka(ctx, record.Headers)
			ctx, span := tracing.StartInfrastructure(ctx, "processMessage", tracing.SubLayerBroker, trace.WithLinks(links...))
			defer span.End()

			if err := kc.processMessage(ctx, record.Value); err != nil {
				log.Printf("Ошибка обработки заказа: %v", err)
			}
		}(ctx)
	}
}

// processMessage отправляет заказ в Temporal
func (kc *KafkaConsumer) processMessage(ctx context.Context, value []byte) error {
	ctx, span := tracing.StartInfrastructure(ctx, "processMessage", tracing.SubLayerBroker)
	defer span.End()

	var order domain.Order
	if err := json.Unmarshal(value, &order); err != nil {
		span.RecordError(err)
		return err
	}

	log.Printf("Начинаем обработку заказа %s через Saga", order.ID)
	err := kc.sagaManager.Execute(ctx, order)
	if err != nil {
		span.RecordError(err)
		log.Printf("Ошибка выполнения Saga для заказа %s: %v", order.ID, err)
		return err
	}

	log.Printf("Заказ %s успешно обработан", order.ID)
	return nil
}

func (kc *KafkaConsumer) Close() {
	kc.client.Close()
}
