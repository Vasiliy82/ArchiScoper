# Определение переменных
COMPOSE_FILE=docker-compose.yml
CLICKHOUSE_CLIENT=docker exec -it clickhouse clickhouse-client --host clickhouse
MIGRATIONS_FILE=deployment/migrations.sql
ORDERS ?= 10
BIN_DIR=./bin

# Создание папки bin, если её нет
prepare:
	mkdir -p $(BIN_DIR)

# Запуск инфраструктуры: Kafka, ClickHouse, OTEL Collector
start-infra:
	docker compose up -d kafka clickhouse otel-collector jaeger

# Сборка сервисов
build: prepare
	docker compose build retailer-api retailer-oms
	go build -C trace-analyzer -o ../$(BIN_DIR)/trace-analyzer main.go
	go build -C trace-analyzer-v2 -o ../$(BIN_DIR)/trace-analyzer-v2 main.go

# Запуск всех сервисов после подготовки инфраструктуры
start:
	docker compose up -d

# Выполнение миграций в ClickHouse
migrate:
	docker cp $(MIGRATIONS_FILE) clickhouse:/migrations.sql
	$(CLICKHOUSE_CLIENT) --multiline --queries-file /migrations.sql

# Полный цикл: запуск инфраструктуры, сборка сервисов, запуск и миграция
deploy: start-infra build start migrate

# Остановка всех контейнеров
stop:
	docker compose down

# Очистка данных (удаление volume)
clean:
	docker compose down -v
	rm -rf $(BIN_DIR)

# Логирование сервисов
logs:
	docker compose logs -f

# Перезапуск проекта с очисткой данных и повторным развертыванием
reset: clean deploy

# Отправка одного запроса
order:
	curl -X POST http://localhost:8081/orders -H "Content-Type: application/json" -d '{}'

# Генерация нагрузки (несколько заказов)
load-test:
	@for i in $(shell seq 1 $(ORDERS)); do \
		curl -X POST http://localhost:8081/orders -H "Content-Type: application/json" -d '{}'; \
	done
report:
	./bin/trace-analyzer > ./report1.dot
	./bin/trace-analyzer-v2 > ./report2.dot
	dot -Tpng ./report1.dot -o ./report1.png
	dot -Tpng ./report2.dot -o ./report2.png