# Stage 1: Сборка Go-приложения
FROM golang:1.23 AS build

WORKDIR /app

# # Устанавливаем необходимые пакеты для сборки
# RUN apt-get update && apt-get install -y \
#     build-essential \
#     libssl-dev \
#     pkg-config

# Копируем go.mod и go.sum (включая зависимости от retailer-api/pkg)
COPY retailer-api/go.mod retailer-api/go.sum retailer-api/
COPY retailer-mockservice/go.mod retailer-mockservice/go.sum retailer-mockservice/

# Обрабатываем зависимости
WORKDIR /app/retailer-mockservice
RUN go mod download

# Копируем исходный код (оба проекта в один GOPATH)
WORKDIR /app
COPY . .

# Проверяем зависимости
WORKDIR /app/retailer-mockservice
RUN go mod verify

# Собираем бинарник
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o /build/retailer-mockservice ./cmd/main.go

# Stage 2: Чистый образ для выполнения
FROM alpine:latest AS backend
WORKDIR /app

# Устанавливаем зависимости (если нужно)
RUN apk add --no-cache libc6-compat

# Копируем собранный бинарник
COPY --from=build /build/retailer-mockservice ./retailer-mockservice

# Указываем команду запуска
CMD ["./retailer-mockservice"]
