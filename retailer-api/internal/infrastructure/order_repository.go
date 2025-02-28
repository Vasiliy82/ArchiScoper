package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/domain"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
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

	// Проверяем, существует ли топик
	topicMetadata, err := r.admin.ListTopics(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения списка топиков: %w", err)
	}

	if _, exists := topicMetadata[topic]; exists {
		log.Printf("Топик %s уже существует", topic)
		return nil
	}

	// Создаем топик
	_, err = r.admin.CreateTopics(ctx, 1, 1, map[string]*string{
		"min.insync.replicas": toPtr("1"), // Гарантируем запись при наличии 1 реплики
	}, topic)

	if err != nil {
		return fmt.Errorf("ошибка создания топика %s: %w", topic, err)
	}

	log.Printf("Топик %s успешно создан", topic)
	return nil
}

// Вспомогательная функция для преобразования строки в *string
func toPtr(val string) *string {
	return &val
}

// PublishOrder отправляет заказ в Kafka
func (r *OrderRepository) PublishOrder(ctx context.Context, order domain.Order) error {
	tracer := otel.Tracer("repository")
	ctx, span := tracer.Start(ctx, "PublishOrder")
	defer span.End()

	// Сериализуем заказ в JSON
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	// Формируем Kafka-сообщение
	record := &kgo.Record{
		Topic: r.topic,
		Key:   []byte(order.ID),
		Value: data,
	}

	// Отправляем сообщение в Kafka с ожиданием подтверждения записи
	err = r.client.ProduceSync(ctx, record).FirstErr()
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения в Kafka: %w", err)
	}

	log.Printf("Сообщение отправлено в Kafka (топик: %s, key: %s)", r.topic, order.ID)
	return nil
}

// Close закрывает Kafka-клиент
func (r *OrderRepository) Close() {
	r.client.Close()
}
