Инфраструктурный слой (Infrastructure Layer)**
   - Отвечает за работу с внешними системами (БД, Kafka, HTTP-клиенты).
   - Имплементирует интерфейсы, определённые в **Use Case**.
   - **Файлы:** `order_repository.go`, `kafka_producer.go`, `postgres_order_storage.go`.
   - **Папка:** `infrastructure/`


 infrastructure/     # Взаимодействие с БД, Kafka, API
 ├── repository/     # БД-репозитории
 ├── eventbus/       # Kafka/SQS и другие очереди
 ├── externalapi/    # HTTP-клиенты

