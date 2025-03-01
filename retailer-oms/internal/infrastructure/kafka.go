package infrastructure

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/Vasiliy82/ArchiScoper/retailer-oms/internal/workflows"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.temporal.io/sdk/client"
)

// KafkaConsumer отвечает за получение заказов из Kafka
type KafkaConsumer struct {
	client      *kgo.Client
	topic       string
	temporalCli client.Client
}

// NewKafkaConsumer создает нового Kafka-консьюмера
func NewKafkaConsumer(brokers []string, topic string, temporalCli client.Client) *KafkaConsumer {
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
		temporalCli: temporalCli,
	}
}

// StartListening запускает обработку сообщений
func (kc *KafkaConsumer) StartListening(ctx context.Context) {
	for {
		fetches := kc.client.PollFetches(ctx)
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			if err := kc.processMessage(ctx, record.Value); err != nil {
				log.Printf("Ошибка обработки заказа: %v", err)
			}
		}
	}
}

// processMessage отправляет заказ в Temporal
func (kc *KafkaConsumer) processMessage(ctx context.Context, value []byte) error {
	ctx, span := tracing.StartInfrastructure(ctx, "ProcessOrder", tracing.SubLayerBroker)
	defer span.End()

	var order domain.Order
	if err := json.Unmarshal(value, &order); err != nil {
		span.RecordError(err)
		return err
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "order-workflow-" + order.ID,
		TaskQueue: "order-tasks",
	}

	_, err := kc.temporalCli.ExecuteWorkflow(ctx, workflowOptions, workflows.OrderWorkflow, order)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (kc *KafkaConsumer) Close() {
	kc.client.Close()
}
