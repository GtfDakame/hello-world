# 🤖 Android Client - Jetpack Compose

## Требования
- Android 8.0+ (API 26)
- Kotlin 1.9+
- Android Studio Hedgehog+

## Архитектура

```
android/
├── app/
│   ├── src/main/
│   │   ├── java/com/messenger/
│   │   │   ├── ui/              # Compose UI
│   │   │   │   ├── theme/
│   │   │   │   ├── components/
│   │   │   │   └── screens/
│   │   │   ├── data/            # Data layer
│   │   │   │   ├── repository/
│   │   │   │   ├── local/       # Room database
│   │   │   │   └── remote/      # API & WebSocket
│   │   │   ├── domain/          # Business logic
│   │   │   │   ├── model/
│   │   │   │   └── usecase/
│   │   │   ├── di/              # Dependency injection (Hilt)
│   │   │   └── util/            # Helpers
│   │   ├── res/                 # Resources
│   │   └── AndroidManifest.xml
│   └── build.gradle.kts
└── gradle/
```

## Tech Stack

- **UI**: Jetpack Compose + Material 3
- **Architecture**: MVVM + Clean Architecture
- **DI**: Hilt
- **Database**: Room
- **Network**: OkHttp + WebSockets
- **Async**: Kotlin Coroutines + Flow
- **Crypto**: Tink / libsodium

## Ключевые компоненты

### 1. Repository Pattern
```kotlin
interface MessageRepository {
    fun getMessages(chatId: String): Flow<List<Message>>
    suspend fun sendMessage(chatId: String, content: String): Result<String>
    suspend fun deleteMessage(messageId: String): Result<Unit>
}
```

### 2. Use Cases
```kotlin
class SendMessageUseCase @Inject constructor(
    private val repo: MessageRepository,
    private val crypto: CryptoService
) {
    suspend operator fun invoke(chatId: String, text: String): Result<String>
}
```

### 3. Compose UI
```kotlin
@Composable
fun ChatScreen(viewModel: ChatViewModel = hiltViewModel()) {
    val uiState by viewModel.uiState.collectAsState()
    
    LazyColumn {
        items(uiState.messages) { message ->
            MessageItem(message)
        }
    }
}
```

### 4. Local Database (Room)
```kotlin
@Entity(tableName = "messages")
data class MessageEntity(
    @PrimaryKey val id: String,
    val chatId: String,
    val senderId: String,
    val content: String?,
    val encryptedContent: ByteArray?,
    val timestamp: Long,
    val isRead: Boolean
)
```

## Сборка

```bash
cd clients/android
./gradlew assembleDebug
```

## Тестирование

```bash
# Unit tests
./gradlew test

# Instrumented tests
./gradlew connectedAndroidTest

# Detekt (code quality)
./gradlew detekt
```

## UI Принципы

- **Material 3**: Dynamic color, adaptive layouts
- **Edge-to-edge**: Immersive режим
- **Predictive Back**: Анимации возврата
- **Accessibility**: TalkBack, достаточный контраст
- **Dark Theme**: Полная поддержка темной темы

## Безопасность

- Keys в Android Keystore (StrongBox если доступен)
- Biometric auth (Fingerprint/Face unlock)
- Root detection
- SSL pinning
- Encrypted SharedPreferences

## Performance Targets

| Метрика | Target |
|---------|--------|
| Cold start | <2s |
| Time to interactive | <1.5s |
| Frame drops | <2% |
| Memory footprint | <150MB |
