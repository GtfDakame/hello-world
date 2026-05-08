# 🌐 Web Client - PWA

## Требования
- Node.js 18+
- Современный браузер (Chrome, Firefox, Safari, Edge)

## Архитектура

```
web/
├── src/
│   ├── components/          # React компоненты
│   ├── pages/               # Страницы
│   ├── hooks/               # Custom hooks
│   ├── services/            # API & WebSocket
│   ├── store/               # Zustand state
│   ├── utils/               # Helpers
│   ├── crypto/              # WebAssembly crypto
│   └── App.tsx
├── public/
│   ├── manifest.json        # PWA manifest
│   └── sw.js                # Service Worker
├── index.html
└── package.json
```

## Tech Stack

- **Framework**: React 18 + TypeScript
- **Build**: Vite
- **State**: Zustand
- **Styling**: Tailwind CSS + Headless UI
- **Crypto**: WebAssembly (Rust → WASM)
- **PWA**: Workbox, Background Sync

## Установка

```bash
cd clients/web

# Install dependencies
npm install

# Development
npm run dev

# Production build
npm run build

# Preview production
npm run preview
```

## PWA Возможности

### 1. Offline Support
```javascript
// Service Worker with Background Sync
self.addEventListener('sync', (event) => {
  if (event.tag === 'send-messages') {
    event.waitUntil(sendPendingMessages())
  }
})
```

### 2. Install Prompt
```typescript
let deferredPrompt: BeforeInstallPromptEvent

window.addEventListener('beforeinstallprompt', (e) => {
  deferredPrompt = e as BeforeInstallPromptEvent
  showInstallButton()
})
```

### 3. Push Notifications
```typescript
async function subscribeToPush() {
  const registration = await navigator.serviceWorker.ready
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: VAPID_PUBLIC_KEY
  })
  // Send subscription to backend
}
```

## WebAssembly Crypto

```rust
// crypto/src/lib.rs (compiled to WASM)
#[wasm_bindgen]
pub fn encrypt_message(data: &[u8], key: &[u8]) -> Vec<u8> {
    // ChaCha20-Poly1305 encryption
}

#[wasm_bindgen]
pub fn decrypt_message(ciphertext: &[u8], key: &[u8]) -> Result<Vec<u8>, JsValue> {
    // Decryption
}
```

```typescript
// TypeScript usage
import init, * as wasm from '../pkg/messenger_crypto.js'

await init()
const encrypted = wasm.encrypt_message(data, key)
```

## Производительность

| Метрика | Target |
|---------|--------|
| First Contentful Paint | <1.5s |
| Time to Interactive | <3s |
| Bundle size (gzipped) | <200KB |
| Lighthouse score | ≥95 |

## Безопасность

- HTTPS only
- CSP headers
- Subresource Integrity (SRI)
- Secure context для крипто
- HttpOnly cookies для сессий

## Браузерная поддержка

- ✅ Chrome 90+
- ✅ Firefox 90+
- ✅ Safari 15+
- ✅ Edge 90+

## Deployment

```bash
# Build
npm run build

# Deploy to CDN / static hosting
# Options: Vercel, Netlify, Cloudflare Pages, S3 + CloudFront
```

## Progressive Enhancement

1. **Core**: Text messaging works everywhere
2. **Enhanced**: Media, reactions on modern browsers
3. **Advanced**: E2EE, PWA features with WASM support
