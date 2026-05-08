# 🚀 Быстрый старт (Quick Start)

## Требования

- Docker & Docker Compose v2.0+
- Node.js 18+ (для разработки веб-клиента)
- Go 1.21+ (для разработки бэкенда)

## Запуск через Docker Compose

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd messenger
```

### 2. Настройка переменных окружения

```bash
cp infra/docker/.env.example infra/docker/.env
# Отредактируйте .env файл, заменив пароли на безопасные значения
```

### 3. Запуск всех сервисов

```bash
cd infra/docker
docker-compose up -d
```

Сервисы будут доступны по адресам:
- **Web клиент**: http://localhost
- **API**: http://localhost:8080
- **WebSocket**: ws://localhost:8080/ws
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### 4. Проверка статуса

```bash
docker-compose ps
docker-compose logs -f backend
```

### 5. Остановка сервисов

```bash
docker-compose down
# Для удаления данных (осторожно!):
docker-compose down -v
```

## Разработка

### Backend

```bash
cd backend

# Установка зависимостей
go mod download

# Запуск сервера в режиме разработки
go run cmd/server/main.go

# Запуск тестов
go test -race ./...

# Сборка бинарного файла
go build -o messenger-server ./cmd/server/main.go
```

### Web Client

```bash
cd clients/web

# Установка зависимостей
npm install

# Запуск dev-сервера
npm run dev

# Сборка для production
npm run build

# Запуск E2E тестов
npm run test:e2e
```

## Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `DB_USER` | Пользователь PostgreSQL | `messenger` |
| `DB_PASSWORD` | Пароль PostgreSQL | _требуется изменить_ |
| `DB_NAME` | Имя базы данных | `messenger` |
| `REDIS_PASSWORD` | Пароль Redis | _требуется изменить_ |
| `JWT_SECRET` | Секретный ключ JWT | _требуется изменить_ |
| `API_PORT` | Порт API | `8080` |
| `HTTP_PORT` | HTTP порт Nginx | `80` |

## Структура проекта

```
messenger/
├── backend/              # Go backend
│   ├── cmd/server/       # Точка входа
│   └── internal/         # Внутренние пакеты
│       ├── api/          # REST API handlers
│       ├── auth/         # Аутентификация
│       ├── chat/         # Чаты и сообщения
│       ├── crypto/       # Криптография
│       ├── database/     # PostgreSQL
│       ├── media/        # Медиа файлы
│       ├── websocket/    # WebSocket hub
│       └── observability/# Metrics & tracing
├── clients/
│   ├── web/              # React/PWA клиент
│   │   ├── src/
│   │   │   ├── components/
│   │   │   ├── pages/
│   │   │   ├── services/
│   │   │   ├── store/
│   │   │   └── styles/
│   │   └── tests/e2e/
│   ├── ios/              # iOS клиент (планируется)
│   ├── android/          # Android клиент (планируется)
│   └── desktop/          # Desktop клиент (планируется)
├── infra/
│   └── docker/           # Docker конфигурации
├── docs/
│   ├── PRD_v1.0.md       # Product Requirements
│   └── adr/              # Architecture Decision Records
└── tests/
    └── e2e/              # E2E тесты
```

## API Документация

После запуска сервера документация доступна по адресу:
- **Swagger UI**: http://localhost:8080/swagger/index.html (после добавления swagger)
- **Health Check**: http://localhost:8080/health
- **Metrics**: http://localhost:8080/metrics

## Тестирование

### Unit тесты

```bash
# Backend
cd backend && go test -race -cover ./...

# Frontend
cd clients/web && npm run test
```

### E2E тесты

```bash
cd clients/web
npm run test:e2e
```

### Проверка безопасности

```bash
# Backend security scan
gosec ./backend/...

# Frontend audit
cd clients/web && npm audit
```

## Troubleshooting

### Ошибка подключения к базе данных

```bash
docker-compose logs postgres
# Убедитесь, что БД запустилась и принимает соединения
```

### Ошибка портов

Если порты заняты, измените их в `.env` файле:
```
API_PORT=8081
HTTP_PORT=8080
```

### Сброс данных

```bash
docker-compose down -v
docker-compose up -d
```

## Лицензия

Проект распространяется под лицензией MIT.

---

**Поддержка:** [GitHub Issues](https://github.com/your-org/messenger/issues)  
**Документация:** [docs/](./docs/)
