# 🖥️ Desktop Client - Tauri v2

## Требования
- Node.js 18+
- Rust 1.70+
- pnpm / npm

## Архитектура

```
desktop/
├── src/
│   ├── main.rs              # Rust backend entry
│   ├── lib.rs               # Tauri commands
│   └── crypto/              # Native crypto (Rust)
├── frontend/
│   ├── src/
│   │   ├── components/      # React/Vue components
│   │   ├── stores/          # State management
│   │   ├── services/        # API & WebSocket
│   │   └── App.tsx
│   ├── index.html
│   └── package.json
├── tauri.conf.json
└── Cargo.toml
```

## Tech Stack

- **Backend**: Rust + Tauri v2
- **Frontend**: React 18 + TypeScript + Vite
- **State**: Zustand / Redux Toolkit
- **Styling**: Tailwind CSS
- **Crypto**: Rust ring / x25519-dalek
- **DB**: SQLite (rusqlite)

## Ключевые возможности

### 1. Tauri Commands (Rust → JS)
```rust
#[tauri::command]
async fn send_message(
    chat_id: String,
    content: String,
    state: State<'_, AppState>
) -> Result<String, String> {
    // Send via WebSocket
}

#[tauri::command]
fn encrypt_message(data: Vec<u8>, key: Vec<u8>) -> Vec<u8> {
    // Native encryption
}
```

### 2. Frontend Service
```typescript
class MessengerService {
  async sendMessage(chatId: string, content: string) {
    return invoke('send_message', { chatId, content })
  }
  
  onMessage(callback: (msg: Message) => void) {
    listen('message-received', callback)
  }
}
```

### 3. Local Database (SQLite)
```sql
CREATE TABLE messages (
  id TEXT PRIMARY KEY,
  chat_id TEXT NOT NULL,
  sender_id TEXT NOT NULL,
  content_encrypted BLOB,
  timestamp INTEGER NOT NULL,
  is_read BOOLEAN DEFAULT FALSE
);
```

## Сборка

```bash
cd clients/desktop

# Install dependencies
pnpm install

# Development
pnpm tauri dev

# Production build
pnpm tauri build
```

## Платформы

- ✅ macOS (Universal: Intel + Apple Silicon)
- ✅ Windows (x64, arm64)
- ✅ Linux (AppImage, deb, rpm)

## Преимущества Tauri

| Характеристика | Tauri | Electron |
|---------------|-------|----------|
| Bundle size | ~15MB | ~150MB |
| Memory usage | ~50MB | ~200MB+ |
| Security | Native isolation | Web context |
| Performance | Native Rust | JavaScript |

## Безопасность

- CSP (Content Security Policy)
- IPC isolation
- Native keychain integration
- Auto-updates с подписью

## Интеграция с ОС

- System tray
- Native notifications
- Global shortcuts
- File system access
- Auto-launch
