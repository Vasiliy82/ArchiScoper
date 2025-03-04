# Stage 1: Сборка Go-приложения
FROM golang:1.23 AS build

WORKDIR /app

# Копируем go.mod и go.sum
COPY retailer-api/go.mod retailer-api/go.sum retailer-api/

# Обрабатываем зависимости
WORKDIR /app/retailer-api
RUN go mod download

# Копируем исходный код
WORKDIR /app
COPY retailer-api ./retailer-api

# Проверяем зависимости
WORKDIR /app/retailer-api
RUN go mod verify

# Собираем бинарник
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o /build/retailer-api ./cmd/main.go

# Stage 2: Чистый образ для выполнения
FROM alpine:latest AS backend
WORKDIR /app

# Устанавливаем зависимости (если нужно)
RUN apk add --no-cache libc6-compat

# Копируем собранный бинарник
COPY --from=build /build/retailer-api ./retailer-api

EXPOSE 8080

# Указываем команду запуска
CMD ["./retailer-api"]
