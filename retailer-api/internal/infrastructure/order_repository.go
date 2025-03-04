package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
)

// OrderRepository отвечает за публикацию заказов в Kafka
type OrderRepository struct {
	client *kgo.Client
	topic  string
	admin  *kadm.Client
}

// NewOrderRepository создает экземпляр OrderRepository с Kafka-клиентом
func NewOrderRepository(brokers []string, topic string) *OrderRepository {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.DefaultProduceTopic(topic),
		kgo.ProduceRequestTimeout(10 * time.Second),
		kgo.RequiredAcks(kgo.AllISRAcks()), // Ждем, пока все реплики подтвердят запись
		kgo.ClientID("retailer-api"),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatalf("Ошибка инициализации Kafka-клиента: %v", err)
	}

	// Административный клиент для управления топиками
	admin := kadm.NewClient(client)

	repo := &OrderRepository{
		client: client,
		topic:  topic,
		admin:  admin,
	}

	// Автоматически создаем топик, если его нет
	if err := repo.EnsureTopicExists(topic); err != nil {
		log.Fatalf("Ошибка создания топика %s: %v", topic, err)
	}

	return repo
}

// EnsureTopicExists проверяет наличие топика и создает его, если он отсутствует
func (r *OrderRepository) EnsureTopicExists(topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx, span := tracing.StartInfrastructure(ctx, "EnsureTopicExists", tracing.SubLayerBroker)
	defer span.End()

	topicMetadata, err := r.admin.ListTopics(ctx)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("ошибка получения списка топиков: %w", err)
	}

	if _, exists := topicMetadata[topic]; exists {
		log.Printf("Топик %s уже существует", topic)
		span.SetAttributes(attribute.Bool("topic.exists", true))
		return nil
	}

	_, err = r.admin.CreateTopics(ctx, 1, 1, map[string]*string{
		"min.insync.replicas": toPtr("1"),
	}, topic)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("ошибка создания топика %s: %w", topic, err)
	}

	span.SetAttributes(attribute.Bool("topic.created", true))
	log.Printf("Топик %s успешно создан", topic)
	return nil
}

// Вспомогательная функция для преобразования строки в *string
func toPtr(val string) *string {
	return &val
}

// PublishOrder отправляет заказ в Kafka
func (r *OrderRepository) PublishOrder(ctx context.Context, order domain.Order) error {
	ctx, span := tracing.StartInfrastructure(ctx, "PublishOrder", tracing.SubLayerBroker)
	defer span.End()

	headers := tracing.InjectTraceContextToKafka(ctx)

	data, err := json.Marshal(order)
	if err != nil {
		span.RecordError(err)
		return err
	}

	record := &kgo.Record{
		Topic:   r.topic,
		Key:     []byte(order.ID),
		Value:   data,
		Headers: headers,
	}

	err = r.client.ProduceSync(ctx, record).FirstErr()
	if err != nil {
		span.RecordError(err)
		log.Printf("Сообщение отправлено в Kafka (топик: %s, key: %s)", r.topic, order.ID)
	}

	span.SetAttributes(attribute.String("kafka.topic", r.topic))
	span.SetAttributes(attribute.String("order.id", order.ID))
	log.Printf("Сообщение отправлено в Kafka (топик: %s, key: %s)", r.topic, order.ID)
	return nil
}

// Close закрывает Kafka-клиент
func (r *OrderRepository) Close() {
	r.client.Close()
}
