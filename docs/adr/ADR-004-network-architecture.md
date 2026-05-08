# ADR-004: Сетевая архитектура и протоколы

**Статус:** Proposed  
**Дата:** 2024-01-15  
**Автор:** Backend Architect  
**Обсуждение:** [GitHub Discussion #4](#)

---

## Контекст

Мессенджер требует надежной, быстрой и безопасной сетевой коммуникации:
- Мгновенная доставка сообщений (<100мс p95)
- Работа в нестабильных сетях (3G, общественный WiFi)
- Эффективная работа с батареей
- Обход блокировок и фаерволов
- Поддержка мультиплексирования (сообщения, медиа, presence)

## Решение

Принята гибридная архитектура с **WebSocket как primary transport** и **QUIC как fallback**.

### Протоколы

| Сценарий | Протокол | Обоснование |
|----------|----------|-------------|
| Real-time сообщения | WebSocket over TLS 1.3 | Низкая задержка, bidirectional |
| Media upload/download | HTTP/3 (QUIC) | Multiplexing, head-of-line blocking resistance |
| Push уведомления | APNs / FCM | Battery efficiency, reliability |
| Файлообмен | HTTP/3 + Range Requests | Resume support, parallel downloads |
| Service discovery | DNS over HTTPS | Privacy, bypass censorship |

### Архитектура подключения

```
┌─────────────────────────────────────────────────────┐
│                    Client                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  WebSocket  │  │    QUIC     │  │   HTTP/3    │ │
│  │   Manager   │  │   Client    │  │   Client    │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │
│         │                │                │         │
│         └────────────────┼────────────────┘         │
│                          │                          │
│              ┌───────────▼───────────┐             │
│              │   Connection Pool     │             │
│              │   & Health Monitor    │             │
│              └───────────┬───────────┘             │
└──────────────────────────┼──────────────────────────┘
                           │
                     Internet
                           │
                           ▼
┌─────────────────────────────────────────────────────┐
│                  Server Side                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │   Gateway   │  │   Media     │  │    CDN      │ │
│  │   (WS/QUIC) │  │   Server    │  │   Edge      │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │
│         │                │                │         │
│         └────────────────┼────────────────┘         │
│                          │                          │
│              ┌───────────▼───────────┐             │
│              │   Load Balancer       │             │
│              │   (Envoy/NGINX)       │             │
│              └───────────┬───────────┘             │
│                          │                          │
│              ┌───────────▼───────────┐             │
│              │   Message Broker      │             │
│              │   (Redpanda/Kafka)    │             │
│              └───────────────────────┘             │
└─────────────────────────────────────────────────────┘
```

### WebSocket менеджер

#### Подключение и реконнект

```go
type WSManager struct {
    mu           sync.RWMutex
    conn         *websocket.Conn
    url          string
    reconnectCnt int
    maxReconnect int
    backoff      time.Duration
    callbacks    map[string]func([]byte)
}

func NewWSManager(url string) *WSManager {
    return &WSManager{
        url:          url,
        maxReconnect: 10,
        backoff:      100 * time.Millisecond,
        callbacks:    make(map[string]func([]byte)),
    }
}

func (m *WSManager) Connect(ctx context.Context) error {
    for attempt := 0; attempt < m.maxReconnect; attempt++ {
        conn, _, err := websocket.Dial(ctx, m.url, nil)
        if err == nil {
            m.conn = conn
            go m.readLoop()
            go m.pingLoop()
            return nil
        }
        
        // Exponential backoff with jitter
        sleep := m.backoff * time.Duration(1<<attempt)
        sleep += time.Duration(rand.Int63n(int64(sleep/4)))
        
        select {
        case <-time.After(sleep):
            continue
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return errors.New("max reconnection attempts reached")
}

func (m *WSManager) readLoop() {
    for {
        _, msg, err := m.conn.ReadMessage()
        if err != nil {
            m.handleDisconnect()
            return
        }
        
        var envelope Envelope
        if err := json.Unmarshal(msg, &envelope); err != nil {
            continue
        }
        
        if cb, ok := m.callbacks[envelope.Type]; ok {
            go cb(envelope.Payload)
        }
    }
}

func (m *WSManager) pingLoop() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        if err := m.conn.WriteControl(
            websocket.PingMessage, 
            []byte{}, 
            time.Now().Add(5*time.Second),
        ); err != nil {
            m.handleDisconnect()
            return
        }
    }
}
```

#### Формат сообщений

```go
type Envelope struct {
    Type      string          `json:"t"`  // Message type
    ID        string          `json:"i"`  // Unique message ID
    Timestamp int64           `json:"ts"` // Unix timestamp
    Payload   json.RawMessage `json:"p"`  // Type-specific payload
}

// Типы сообщений
const (
    MsgTypeMessage      = "msg"
    MsgTypeDeliveryAck  = "ack"
    MsgTypeTyping       = "typing"
    MsgTypePresence     = "presence"
    MsgTypeSyncRequest  = "sync_req"
    MsgTypeSyncResponse = "sync_res"
    MsgTypeError        = "err"
)

type MessagePayload struct {
    ChatID    string `json:"cid"`
    Content   string `json:"c,omitempty"`
    MediaType string `json:"mt,omitempty"`
    MediaURL  string `json:"mu,omitempty"`
    ReplyTo   string `json:"rt,omitempty"`
}

type DeliveryAck struct {
    MessageID string `json:"mid"`
    Status    string `json:"s"` // delivered, read, failed
    Timestamp int64  `json:"ts"`
}
```

### QUIC реализация

#### Почему QUIC для fallback

```
Преимущества QUIC:
✅ Multiplexing без head-of-line blocking
✅ 0-RTT connection resumption
✅ Better congestion control
✅ UDP-based (обход некоторых блокировок)
✅ Built-in encryption (TLS 1.3)

Когда используется:
- WebSocket заблокирован или нестабилен
- Высокий packet loss (>5%)
- Частые переключения между WiFi/cellular
```

```rust
// Rust QUIC client using quinn
use quinn::{Endpoint, ClientConfig, Connection};

pub struct QuicClient {
    endpoint: Endpoint,
    server_addr: SocketAddr,
}

impl QuicClient {
    pub async fn connect(&self) -> Result<Connection> {
        let conn = self.endpoint
            .connect(self.server_addr, "messenger.example.com")?
            .await?;
        
        Ok(conn)
    }
    
    pub async fn send_message(&self, conn: &Connection, msg: &[u8]) -> Result<()> {
        let mut send = conn.open_uni().await?;
        send.write_all(msg).await?;
        send.finish()?;
        Ok(())
    }
}

fn create_client_config() -> ClientConfig {
    let mut crypto = rustls::ClientConfig::builder()
        .with_safe_defaults()
        .with_root_certificates(root_store)
        .with_no_client_auth();
    
    crypto.alpn_protocols = vec![b"h3".to_vec()];
    
    let mut config = ClientConfig::new(Arc::new(crypto));
    
    // Transport parameters
    let mut transport = quinn::TransportConfig::default();
    transport
        .max_concurrent_uni_streams(100u32.into())
        .max_concurrent_bidi_streams(100u32.into())
        .keep_alive_interval(Some(Duration::from_secs(30)));
    
    config.transport_config(Arc::new(transport));
    config
}
```

### Адаптивный bitrate для медиа

```go
type AdaptiveUploader struct {
    minBandwidth int64
    maxBandwidth int64
    currentBW    int64
    history      []BandwidthSample
}

type BandwidthSample struct {
    timestamp time.Time
    bytes     int64
    duration  time.Duration
}

func (a *AdaptiveUploader) EstimateBandwidth() int64 {
    // Weighted average of recent samples
    now := time.Now()
    var weightedSum float64
    var weightTotal float64
    
    for i := len(a.history) - 1; i >= 0 && i > len(a.history)-10; i-- {
        sample := a.history[i]
        age := now.Sub(sample.timestamp).Seconds()
        weight := 1.0 / (age + 1.0) // Decay older samples
        
        bw := float64(sample.bytes) / sample.duration.Seconds()
        weightedSum += bw * weight
        weightTotal += weight
    }
    
    if weightTotal == 0 {
        return a.minBandwidth
    }
    
    return int64(weightedSum / weightTotal)
}

func (a *AdaptiveUploader) SelectQuality(availableBW int64) Quality {
    switch {
    case availableBW < 500_000: // 500 Kbps
        return QualityLow
    case availableBW < 2_000_000: // 2 Mbps
        return QualityMedium
    case availableBW < 10_000_000: // 10 Mbps
        return QualityHigh
    default:
        return QualityOriginal
    }
}
```

### Connection pooling и health check

```go
type ConnectionPool struct {
    mu       sync.Mutex
    active   map[string]*Connection
    idle     []*Connection
    maxSize  int
    healthCh chan *Connection
}

func (p *ConnectionPool) Get(ctx context.Context) (*Connection, error) {
    p.mu.Lock()
    
    // Try idle pool first
    if len(p.idle) > 0 {
        conn := p.idle[len(p.idle)-1]
        p.idle = p.idle[:len(p.idle)-1]
        p.mu.Unlock()
        
        if p.isHealthy(conn) {
            return conn, nil
        }
        
        // Connection unhealthy, get new one
        return p.createConnection(ctx)
    }
    
    p.mu.Unlock()
    return p.createConnection(ctx)
}

func (p *ConnectionPool) isHealthy(conn *Connection) bool {
    // Check last activity, ping response, error rate
    return time.Since(conn.LastActivity) < 5*time.Minute &&
           conn.ErrorRate < 0.05
}

func (p *ConnectionPool) healthMonitor() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        p.mu.Lock()
        for _, conn := range p.active {
            if !p.isHealthy(conn) {
                p.healthCh <- conn
            }
        }
        p.mu.Unlock()
    }
}
```

### Обработка сетевых изменений

```swift
// iOS Network Reachability
class NetworkMonitor {
    private let monitor = NWPathMonitor()
    private var currentPath: NWPath?
    
    var onNetworkChange: ((NetworkType) -> Void)?
    
    init() {
        monitor.pathUpdateHandler = { [weak self] path in
            self?.currentPath = path
            self?.handlePathUpdate(path)
        }
        monitor.start(queue: DispatchQueue(label: "network-monitor"))
    }
    
    private func handlePathUpdate(_ path: NWPath) {
        guard let onNetworkChange = onNetworkChange else { return }
        
        switch path.status {
        case .satisfied:
            if path.usesInterfaceType(.wifi) {
                onNetworkChange(.wifi)
            } else if path.usesInterfaceType(.cellular) {
                onNetworkChange(.cellular)
            } else {
                onNetworkChange(.other)
            }
            
        case .unsatisfied, .requiresConnection:
            onNetworkChange(.offline)
            
        @unknown default:
            break
        }
    }
    
    enum NetworkType {
        case wifi, cellular, other, offline
    }
}

// Реакция на изменения
extension MessengerService {
    func networkDidChange(to type: NetworkType) {
        switch type {
        case .offline:
            pauseOutgoingQueue()
            enableOfflineMode()
            
        case .cellular:
            reduceMediaQuality()
            limitBackgroundSync()
            
        case .wifi:
            resumeFullSync()
            prefetchContent()
            
        default:
            break
        }
    }
}
```

### Балансировка нагрузки

```yaml
# Envoy configuration snippet
static_resources:
  listeners:
  - name: messenger_gateway
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 443
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          stat_prefix: ingress_http
          codec_type: AUTO
          route_config:
            name: local_route
            virtual_hosts:
            - name: backend
              domains: ["*"]
              routes:
              - match:
                  prefix: "/ws"
                route:
                  cluster: websocket_cluster
              - match:
                  prefix: "/media"
                route:
                  cluster: media_cluster
          
          http_filters:
          - name: envoy.filters.http.health_check
          - name: envoy.filters.http.router

  clusters:
  - name: websocket_cluster
    connect_timeout: 5s
    type: STRICT_DNS
    lb_policy: LEAST_REQUEST
    health_checks:
    - timeout: 5s
      interval: 10s
      unhealthy_threshold: 3
      http_health_check:
        path: /health
```

## Последствия

### Положительные
- ✅ Минимальная задержка доставки
- ✅ Устойчивость к проблемам сети
- ✅ Эффективное использование батареи
- ✅ Обход некоторых типов блокировок

### Отрицательные
- ⚠️ Сложность реализации двух протоколов
- ⚠️ Увеличенный размер бинарников
- ⚠️ Требуется дополнительная инфраструктура

### Нейтральные
- ➖ QUIC требует UDP порты открыты
- ➶ WebSocket требует stable connection

## Альтернативы

### 1. Только HTTP long-polling
**Отклонено:** Высокая задержка, неэффективно

### 2. Только QUIC
**Отложено:** Меньшая совместимость, особенно на старых устройствах

### 3. MQTT
**Отклонено:** Избыточен для чата, проблемы с масштабированием

## План внедрения

1. **Неделя 1-2:** WebSocket базовая реализация
2. **Неделя 3-4:** Reconnection logic & heartbeat
3. **Неделя 5-6:** QUIC client integration
4. **Неделя 7-8:** Adaptive bandwidth algorithm
5. **Неделя 9-10:** Network change handling
6. **Неделя 11-12:** Load testing (100k concurrent connections)
7. **Неделя 13-14:** Battery usage optimization

## Метрики успеха

| Метрика | Target | Измерение |
|---------|--------|-----------|
| Message latency (p95) | <100мс | Send to delivery ack |
| Reconnection time | <1 сек | Disconnect to reconnect |
| Battery drain (background) | <2%/hour | Idle with connection |
| WebSocket upgrade success | ≥99% | Initial connection |
| QUIC fallback trigger | <5% | When WebSocket fails |
| Message delivery rate | ≥99.99% | Successful deliveries |

## Ссылки

- [RFC 6455 (WebSocket)](https://datatracker.ietf.org/doc/html/rfc6455)
- [RFC 9000 (QUIC)](https://datatracker.ietf.org/doc/html/rfc9000)
- [quinn - Rust QUIC](https://github.com/quinn-rs/quinn)
- [Envoy Proxy](https://www.envoyproxy.io/)
- [NWPathMonitor - Apple](https://developer.apple.com/documentation/network/nwpathmonitor)

---

*ADR требует обновления после load testing*
