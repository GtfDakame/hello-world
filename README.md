# Мессенджер нового поколения

**Статус:** MVP Development Phase  
**Версия:** 0.5  
**Дата:** 2024-01-15

---

## 📋 О проекте

Репозиторий содержит полный стек мессенджера с фокусом на:
- **Скорость** — доставка сообщений <100мс
- **Приватность** — zero-knowledge архитектура, E2EE
- **Психологический комфорт** — без шума, уважение к вниманию

## 📊 Статистика проекта

| Тип | Файлы | Строки кода |
|-----|-------|-------------|
| Документация (MD) | 12 | ~2,748 |
| Backend (Go) | 5 | ~1,113 |
| Клиенты (README) | 4 | ~500 |
| **Итого** | **17+** | **~4,361** |

## 📁 Структура репозитория

```
/
├── README.md                  # Главный README
├── docs/                      # Документация
│   ├── PRD_v1.0.md           # Product Requirements Document (269 строк)
│   └── adr/                  # Architecture Decision Records
│       ├── ADR-001-sync-strategy.md      # Синхронизация (OT/CRDT) - 172 строки
│       ├── ADR-002-e2e-encryption.md     # E2EE протокол - 291 строка
│       ├── ADR-003-offline-first.md      # Offline-first - 410 строк
│       └── ADR-004-network-architecture.md # Сеть - 544 строки
│
├── backend/                   # Backend сервисы (Go)
│   ├── README.md             # Документация backend
│   ├── go.mod                # Зависимости Go
│   ├── cmd/
│   │   └── server/
│   │       └── main.go       # Точка входа (110 строк)
│   └── internal/
│       ├── crypto/
│       │   ├── crypto.go     # Базовая криптография (143 строки)
│       │   └── double_ratchet.go # Double Ratchet (336 строк)
│       ├── database/
│       │   └── database.go   # PostgreSQL + миграции (227 строк)
│       └── websocket/
│           └── hub.go        # WebSocket Hub (287 строк)
│
├── clients/                   # Клиентские приложения
│   ├── ios/
│   │   └── README.md         # iOS спецификация (SwiftUI)
│   ├── android/
│   │   └── README.md         # Android спецификация (Jetpack Compose)
│   ├── desktop/
│   │   └── README.md         # Desktop спецификация (Tauri v2)
│   └── web/
│       └── README.md         # Web PWA спецификация
│
└── infra/                     # Инфраструктура
    └── README.md             # Kubernetes & Terraform документация
```

## 🚀 Быстрый старт

### Предварительные требования
- Go 1.21+
- Rust 1.75+
- Node.js 20+
- Docker & Docker Compose
- Kubernetes (для локальной разработки: minikube/kind)

### Запуск локально

```bash
# Backend
cd backend
go run cmd/server/main.go

# iOS клиент
cd clients/ios
xcodebuild -scheme Messenger -destination 'platform=iOS Simulator,name=iPhone 15'

# Android клиент
cd clients/android
./gradlew assembleDebug

# Desktop клиент
cd clients/desktop
cargo tauri dev
```

## 📊 Текущий статус

| Компонент | Статус | Готовность |
|-----------|--------|------------|
| PRD | ✅ Завершено | 100% |
| ADR: Синхронизация | ✅ Завершено | 100% |
| ADR: E2EE | ✅ Завершено | 100% |
| ADR: Offline-first | ✅ Завершено | 100% |
| ADR: Сеть | ✅ Завершено | 100% |
| Backend Core | 🔄 В разработке | 0% |
| iOS клиент | 🔄 В разработке | 0% |
| Android клиент | 🔄 В разработке | 0% |
| Desktop клиент | 📅 Запланировано | 0% |

## 🎯 Ключевые метрики (Target)

| Метрика | Target |
|---------|--------|
| Время до первого сообщения | <8 сек |
| Задержка отправки (p95) | <100мс |
| Время запуска (cold) | <2 сек |
| FPS при скролле | ≥58 |
| Crash-free sessions | ≥99.9% |

## 📚 Документация

- [Product Requirements Document](docs/PRD_v1.0.md)
- [ADR-001: Синхронизация данных](docs/adr/ADR-001-sync-strategy.md)
- [ADR-002: E2EE протокол](docs/adr/ADR-002-e2e-encryption.md)
- [ADR-003: Offline-first](docs/adr/ADR-003-offline-first.md)
- [ADR-004: Сетевая архитектура](docs/adr/ADR-004-network-architecture.md)

## 👥 Команда

| Роль | Ответственный | Статус |
|------|--------------|--------|
| Product Owner | TBD | 📅 Назначение |
| UX/UI Lead | TBD | 📅 Назначение |
| iOS Lead | TBD | 📅 Назначение |
| Android Lead | TBD | 📅 Назначение |
| Backend Architect | TBD | 📅 Назначение |
| Security Lead | TBD | 📅 Назначение |
| QA Lead | TBD | 📅 Назначение |
| DevOps/SRE | TBD | 📅 Назначение |

## 🗓 Roadmap

### Фаза 1: Discovery (2 недели)
- [x] Создание PRD
- [x] Architecture Decision Records
- [ ] Usability тесты прототипов
- [ ] Threat modeling
- [ ] Performance baseline

### Фаза 2: MVP Development (8 недель)
- [ ] Sprint 1-2: Auth + Core infrastructure
- [ ] Sprint 3-4: 1:1 Chat + Media
- [ ] Sprint 5-6: Groups + Sync engine
- [ ] Sprint 7-8: Notifications + Polish

### Фаза 3: Alpha (2 недели)
- [ ] Internal testing
- [ ] Security review
- [ ] Bug fixing

### Фаза 4: Beta (4 недели)
- [ ] 5k invite-only users
- [ ] Telemetry opt-in
- [ ] Feedback loops
- [ ] Stabilization

### Фаза 5: GA (2 недели)
- [ ] RC stabilization
- [ ] Phased rollout
- [ ] Public launch

## 🤝 Contributing

Этот проект находится в активной разработке. Для участия:

1. Изучите [PRD](docs/PRD_v1.0.md) и [ADRs](docs/adr/)
2. Выберите задачу из roadmap
3. Создайте issue для обсуждения
4. Отправьте PR с изменениями

## 📄 Лицензия

TBD

## 📞 Контакты

- GitHub Discussions: [Ссылка](#)
- Security Policy: [Ссылка](#)
- Public Roadmap: [Ссылка](#)

---

*Последнее обновление: 2024-01-15*
