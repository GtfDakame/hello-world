# 📱 iOS Client - SwiftUI

## Требования
- iOS 16.0+
- Xcode 15.0+
- Swift 5.9+

## Архитектура

```
ios/
├── MessengerApp.swift       # Entry point
├── Models/                  # Data models
│   ├── User.swift
│   ├── Chat.swift
│   └── Message.swift
├── Views/                   # SwiftUI views
│   ├── ContentView.swift
│   ├── ChatList/
│   ├── Conversation/
│   └── Settings/
├── ViewModels/              # Business logic
│   ├── AuthViewModel.swift
│   ├── ChatViewModel.swift
│   └── SettingsViewModel.swift
├── Services/                # Network & storage
│   ├── WebSocketService.swift
│   ├── APIService.swift
│   ├── CryptoService.swift
│   └── StorageService.swift
├── Utils/                   # Helpers
└── Resources/               # Assets, locales
```

## Установка зависимостей

Используем Swift Package Manager:

```swift
// Package.dependencies
dependencies: [
    .package(url: "https://github.com/apple/swift-protobuf.git", from: "1.26.0"),
    .package(url: "https://github.com/krzyzanowskim/CryptoSwift.git", from: "1.8.0"),
]
```

## Ключевые компоненты

### 1. WebSocket Service
```swift
class WebSocketService: ObservableObject {
    func connect(userID: String, deviceID: String)
    func sendMessage(chatID: String, content: String)
    func subscribeToChat(_ chatID: String)
}
```

### 2. Crypto Service (E2EE)
```swift
class CryptoService {
    func generateKeyPair() -> KeyPair
    func establishSession(with peerKeys: PreKeys) throws -> Session
    func encrypt(message: Data, session: Session) throws -> EncryptedMessage
    func decrypt(encrypted: EncryptedMessage, session: Session) throws -> Data
}
```

### 3. Local Storage (Core Data)
```swift
class StorageService {
    func saveMessage(_ message: Message)
    func fetchMessages(chatID: String, limit: Int) -> [Message]
    func deleteMessage(_ messageID: UUID)
}
```

## UI Принципы

- **Native HIG**: Следовать Human Interface Guidelines 2024
- **Dynamic Type**: Поддержка всех размеров шрифта
- **Accessibility**: VoiceOver, Reduce Motion, High Contrast
- **Offline-first**: UI всегда отзывчив, синхронизация в фоне

## Сборка

```bash
cd clients/ios
xcodebuild -scheme Messenger -configuration Debug build
```

## Тестирование

```bash
xcodebuild test -scheme Messenger -destination 'platform=iOS Simulator,name=iPhone 15'
```

## Безопасность

- Keys stored in Secure Enclave
- Biometric auth (Face ID / Touch ID)
- Jailbreak detection
- Certificate pinning
