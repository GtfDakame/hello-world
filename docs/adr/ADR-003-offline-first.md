# ADR-003: Offline-first архитектура и кэширование

**Статус:** Proposed  
**Дата:** 2024-01-15  
**Автор:** Backend Architect  
**Обсуждение:** [GitHub Discussion #3](#)

---

## Контекст

Мессенджер должен работать полноценно без соединения:
- Чтение истории сообщений
- Написание и отправка сообщений (queued)
- Поиск по локальным данным
- Просмотр медиа (кэшированное)
- Управление чатами и настройками

При восстановлении соединения требуется:
- Бесшовная синхронизация
- Разрешение конфликтов
- Оптимистичные обновления UI

## Решение

Принята архитектура **Local-First с событийной синхронизацией**.

### Принципы

1. **Единственный источник истины — локальная БД**
   - Сервер — координатор, а не владелец данных
   - Клиент работает всегда, даже offline

2. **Оптимистичный UI**
   - Изменения применяются немедленно
   - Синхронизация происходит в фоне
   - Откат только при критических ошибках

3. **Идемпотентность операций**
   - Все операции имеют уникальные ID
   - Повторное применение безопасно
   - Нет дублирования данных

### Клиентское хранилище

#### iOS: Core Data + SQLite гибридный подход

```swift
// Core Data для реляционных данных
@Model
class Message {
    @Attribute(.unique) let id: String
    var content: String
    var timestamp: Date
    var authorID: String
    var chatID: String
    var syncStatus: SyncStatus // pending, synced, failed
    var vectorClock: Data
}

// SQLite для full-text search
CREATE VIRTUAL TABLE message_fts USING fts5(
    content,
    author_name,
    chat_name
);

// File system для медиа
// Documents/messenger/cache/media/{chat_id}/{message_id}.{ext}
```

#### Android: Room + DataStore

```kotlin
@Entity
data class Message(
    @PrimaryKey val id: String,
    val content: String,
    val timestamp: Long,
    val authorId: String,
    val chatId: String,
    val syncStatus: SyncStatus,
    val vectorClock: ByteArray
)

@Dao
interface MessageDao {
    @Query("SELECT * FROM message WHERE chatId = :chatId ORDER BY timestamp DESC")
    fun getByChat(chatId: String): Flow<List<Message>>
    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(message: Message)
    
    @Query("SELECT * FROM message WHERE syncStatus = 'PENDING'")
    suspend fun getPending(): List<Message>
}

// Search integration
@FtsEntity
data class MessageSearchIndex(
    val messageId: String,
    val content: String,
    val metadata: String
)
```

#### Desktop/Web: SQLite + IndexedDB

```rust
// Tauri/Rust core
use rusqlite::Connection;

fn init_db(conn: &Connection) -> Result<()> {
    conn.execute_batch(&[
        "CREATE TABLE IF NOT EXISTS messages (
            id TEXT PRIMARY KEY,
            content TEXT NOT NULL,
            timestamp INTEGER NOT NULL,
            author_id TEXT,
            chat_id TEXT,
            sync_status TEXT DEFAULT 'pending',
            vector_clock BLOB
        )",
        "CREATE INDEX idx_chat_timestamp ON messages(chat_id, timestamp DESC)",
        "CREATE INDEX idx_sync_status ON messages(sync_status)"
    ].join(";"))?;
    Ok(())
}
```

### Стратегия кэширования

#### Уровни кэша

```
┌─────────────────────────────────────────┐
│           Memory Cache (LRU)            │
│ - Текущий чат (последние 100 сообщений) │
│ - Аватарки, превью                      │
│ - Время жизни: до выгрузки из памяти    │
└─────────────────────────────────────────┘
                    ↓ eviction
┌─────────────────────────────────────────┐
│          Disk Cache (SQLite)            │
│ - Полная история чатов                  │
│ - Метаданные, контакты                  │
│ - Время жизни: персистентно             │
└─────────────────────────────────────────┘
                    ↓ prefetch
┌─────────────────────────────────────────┐
│        Media Cache (File System)        │
│ - Фото, видео, документы                │
│ - Лимит: 1GB (настраиваемо)             │
│ - LRU eviction по размеру               │
└─────────────────────────────────────────┘
```

#### Политики кэширования

| Тип данных | Стратегия | TTL | Лимит |
|------------|-----------|-----|-------|
| Текстовые сообщения | Cache-first | ∞ | Без лимита |
| Медиа (фото) | Cache-first, network fallback | 30 дней | 500MB |
| Медиа (видео) | Network-first, cache on play | 7 дней | 200MB |
| Аватарки | Stale-while-revalidate | 24 часа | 50MB |
| Профили пользователей | Cache-first | 7 дней | 10MB |
| Настройки | Cache-first | ∞ | - |

### Очередь исходящих операций

```go
type OutgoingQueue struct {
    mu sync.Mutex
    queue []*Operation
    db    *sql.DB
}

type Operation struct {
    ID        string    // UUID v4
    Type      OpType    // SEND_MESSAGE, EDIT, DELETE, REACTION
    Payload   []byte    // JSON serialized
    RetryCount int
    CreatedAt time.Time
    Status    OpStatus  // PENDING, PROCESSING, COMPLETED, FAILED
}

func (q *OutgoingQueue) Enqueue(op *Operation) error {
    q.mu.Lock()
    defer q.mu.Unlock()
    
    // Сохраняем в SQLite
    _, err := q.db.Exec(`
        INSERT INTO outgoing_queue 
        (id, type, payload, retry_count, created_at, status)
        VALUES (?, ?, ?, 0, ?, 'PENDING')
    `, op.ID, op.Type, op.Payload, op.CreatedAt)
    
    // Триггерим background sync
    go q.ProcessQueue()
    
    return err
}

func (q *OutgoingQueue) ProcessQueue() {
    // Exponential backoff
    // Concurrent limit: 5 operations
    // Idempotency keys для сервера
}
```

### Синхронизация при reconnect

```
┌──────────────┐
│  Connection  │
│   Restored   │
└──────┬───────┘
       │
       ▼
┌──────────────┐     ┌──────────────┐
│ Delta Sync   │ ──► │ Pending Ops  │
│ Request      │     │ Queue        │
└──────┬───────┘     └──────┬───────┘
       │                    │
       ▼                    ▼
┌──────────────┐     ┌──────────────┐
│ Apply Remote │     │ Send Local   │
│ Changes      │     │ Operations   │
└──────┬───────┘     └──────┬───────┘
       │                    │
       └────────┬───────────┘
                │
                ▼
       ┌────────────────┐
       │ Conflict       │
       │ Resolution     │
       └────────┬───────┘
                │
                ▼
       ┌────────────────┐
       │ UI Update      │
       │ (Reactive)     │
       └────────────────┘
```

### Протокол delta sync

```
Client → Server: DELTA_SYNC {
    vector_clock: {...},
    last_message_id: "...",
    chat_ids: ["...", "..."]
}

Server → Client: DELTA_RESPONSE {
    new_vector_clock: {...},
    messages: [...],
    deleted_ids: [...],
    updated_metadata: {...}
}

Client: Apply changes, update local DB, notify UI
```

### Обработка конфликтов

```rust
enum ConflictResolution {
    // Для текста: merge через OT
    MergeText {
        local_ops: Vec<Operation>,
        remote_ops: Vec<Operation>,
    },
    
    // Для метаданных: last-write-wins с vector clock
    LastWriteWins {
        local_vc: VectorClock,
        remote_vc: VectorClock,
    },
    
    // Для удаления: tombstone propagation
    Tombstone {
        deletion_time: i64,
        device_id: String,
    },
}

fn resolve_conflict(conflict: Conflict) -> ResolvedState {
    match conflict.r#type {
        MessageType::Text => apply_ot(conflict.local, conflict.remote),
        MessageType::Metadata => lww_resolve(conflict),
        MessageType::Deletion => propagate_tombstone(conflict),
    }
}
```

### Префетчинг и предиктивная загрузка

```swift
// Предсказание следующего действия пользователя
class PredictivePrefetcher {
    
    func predictAndPrefetch(userActivity: UserActivity) {
        switch userActivity {
        case .openingChat(let chatId):
            prefetchMessages(chatId, count: 100)
            prefetchMedia(chatId, types: [.image])
            prefetchUserProfiles(chatId)
            
        case .scrollingToTop(let chatId):
            prefetchOlderMessages(chatId, count: 50)
            
        case .searchQuery(let query):
            prefetchSearchResults(query)
            
        default:
            break
        }
    }
    
    private func prefetchMessages(_ chatId: String, count: Int) {
        // Check cache first
        // If partial, request delta from server
        // Store in memory cache for instant access
    }
}
```

### Миграция и бэкап

```
Бэкап стратегии:

1. Encrypted Cloud Backup (opt-in)
   - End-to-end encrypted archive
   - User holds backup key
   - Incremental backups

2. Local Export
   - JSON/HTML export
   - Include media option
   - Password protection

3. Device-to-Device Transfer
   - QR code pairing
   - Direct WiFi transfer
   - Encrypted channel
```

## Последствия

### Положительные
- ✅ Мгновенный отклик UI (нет network latency)
- ✅ Работа в любых условиях связи
- ✅ Снижение нагрузки на сервер
- ✅ Лучший UX (no spinners)

### Отрицательные
- ⚠️ Увеличенное использование диска
- ⚠️ Сложность синхронизации
- ⚠️ Риск рассинхронизации
- ⚠️ Трудности отладки

### Нейтральные
- ➖ Требуется миграция данных при изменении схемы
- ➶ Необходимость управления размером кэша

## Альтернативы

### 1. Network-first с кэшем
**Отклонено:** Плохой offline UX, задержки

### 2. Полный P2P без сервера
**Отклонено:** Проблемы с доставкой, NAT traversal

### 3. Server-authoritative
**Отклонено:** Нарушает offline-first принцип

## План внедрения

1. **Неделя 1-2:** Schema design для SQLite/Room/Core Data
2. **Неделя 3-4:** Repository pattern implementation
3. **Неделя 5-6:** Outgoing queue & retry logic
4. **Неделя 7-8:** Delta sync protocol
5. **Неделя 9-10:** Conflict resolution testing
6. **Неделя 11-12:** Performance optimization
7. **Неделя 13-14:** Battery & data usage profiling

## Метрики успеха

| Метрика | Target | Измерение |
|---------|--------|-----------|
| App start (cold) | <2 сек | Cold launch to interactive |
| Message render | <16мс | From tap to display |
| Offline operations | 100% | Functional without network |
| Sync completion | <5 сек | After reconnect |
| Cache hit rate | ≥90% | For repeated queries |
| Disk usage growth | <50MB/day | Average user |

## Ссылки

- [Offline First Architecture](https://offlinefirst.org/)
- [CRDT for Collaborative Apps](https://crdt.tech/)
- [SQLite Best Practices](https://www.sqlite.org/bestpractices.html)
- [Room Persistence Library](https://developer.android.com/training/data-storage/room)
- [Core Data Documentation](https://developer.apple.com/documentation/coredata)

---

*ADR требует обновления после performance testing*
