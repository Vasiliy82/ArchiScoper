### Интерфейсный слой (Interfaces Layer)
- Обрабатывает внешние запросы (HTTP, gRPC, CLI, события Kafka и т. д.).
- Вызывает Use Case для выполнения бизнес-логики.
- Файлы: API-хендлеры, gRPC-серверы, consumers.
- Папка: handler/


 handler/          # Контроллеры: HTTP, gRPC, Kafka
 ├── http/           # REST API хендлеры
 ├── grpc/           # gRPC серверы (если есть)
 ├── consumer/       # Kafka consumers (если есть)