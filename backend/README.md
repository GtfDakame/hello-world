# 🚀 Messenger Backend - Go Implementation

## Структура проекта

```
backend/
├── cmd/
│   └── server/          # Точка входа приложения
├── internal/
│   ├── api/             # REST API handlers
│   ├── auth/            # Аутентификация и авторизация
│   ├── chat/            # Логика чатов
│   ├── crypto/          # Криптография (E2EE, Double Ratchet)
│   ├── database/        # Работа с PostgreSQL
│   ├── media/           # Обработка медиа
│   ├── queue/           # Очереди сообщений (Kafka/Redpanda)
│   ├── sync/            # Синхронизация (CRDT/OT)
│   └── websocket/       # WebSocket hub
└── go.mod               # Зависимости Go
```

## Запуск

### Требования
- Go 1.21+
- PostgreSQL 15+
- Redis 7+

### Локальный запуск

```bash
# Установка зависимостей
cd backend
go mod download

# Переменные окружения
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=messenger
export DB_PASSWORD=your_password
export DB_NAME=messenger

# Запуск сервера
go run cmd/server/main.go -port 8080
```

### Docker Compose (рекомендуется)

```bash
docker-compose up -d postgres redis
go run cmd/server/main.go
```

## API Endpoints

### WebSocket
- `ws://localhost:8080/ws?user_id={uuid}&device_id={device}` - Подключение к WebSocket

### HTTP
- `GET /health` - Проверка здоровья сервиса

## Типы сообщений WebSocket

| Тип | Описание |
|-----|----------|
| `chat` | Отправка/получение сообщения |
| `ack` | Подтверждение доставки |
| `sync` | Синхронизация истории |
| `presence` | Статус онлайн/офлайн |
| `typing` | Индикатор набора текста |
| `reaction` | Реакции на сообщения |
| `error` | Сообщения об ошибках |

## Криптография

Реализовано в `internal/crypto/`:

- **X25519** - Генерация ключей
- **X3DH** - Протокол согласования ключей
- **Double Ratchet** - Постепенное обновление ключей
- **AES-256-GCM** / **ChaCha20-Poly1305** - Шифрование сообщений

## База данных

Схема автоматически создается при первом запуске (миграции в `database.go`):

- `users` - Пользователи
- `sessions` - Сессии устройств
- `chats` - Чаты (direct/group/channel)
- `messages` - Сообщения с CRDT метаданными
- `e2ee_sessions` - Сессии E2EE
- `pre_keys` - Pre-keys для X3DH

## Мониторинг

Встроенная поддержка:
- Zap Logger (структурированное логирование)
- Prometheus metrics (готовность к интеграции)
- Health check endpoint

## Безопасность

- ✅ Zero-knowledge архитектура
- ✅ E2EE по умолчанию
- ✅ Forward secrecy (Double Ratchet)
- ✅ Secure key storage (Keychain/Keystore на клиентах)

## Следующие шаги

1. [ ] Реализовать REST API для авторизации
2. [ ] Добавить интеграцию с Redis для кэширования
3. [ ] Реализовать очередь сообщений (Kafka)
4. [ ] Добавить unit/integration тесты
5. [ ] Настроить CI/CD pipeline

## Лицензия

Proprietary - Все права защищены
