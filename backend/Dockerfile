# Multi-stage build для минимального размера образа
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Копирование go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка бинарного файла с оптимизациями
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /messenger-server ./cmd/server/main.go

# Финальный образ
FROM alpine:3.19

WORKDIR /app

# Установка часового пояса и корневых сертификатов
RUN apk --no-cache add ca-certificates tzdata && \
    update-ca-certificates

# Создание пользователя без прав root для безопасности
RUN addgroup -g 1000 messenger && \
    adduser -D -u 1000 -G messenger messenger

# Копирование бинарного файла из builder
COPY --from=builder /messenger-server /app/messenger-server

# Создание директории для медиафайлов
RUN mkdir -p /app/media && chown -R messenger:messenger /app

# Переключение на пользователя без root прав
USER messenger

# Порт API
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/health || exit 1

# Запуск сервера
CMD ["/app/messenger-server"]
