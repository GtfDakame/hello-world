# ADR-002: End-to-End Encryption протокол

**Статус:** Proposed  
**Дата:** 2024-01-15  
**Автор:** Security Lead  
**Обсуждение:** [GitHub Discussion #2](#)

---

## Контекст

Требуется реализовать сквозное шифрование для приватных чатов с гарантиями:
- **Forward Secrecy:** Компрометация ключа не раскрывает прошлые сообщения
- **Deniable Authentication:** Невозможно криптографически доказать авторство третьей стороне
- **Post-Compromise Security:** Автоматическое восстановление безопасности после компрометации
- **Multi-device:** Синхронизация между устройствами без ущерба безопасности

## Решение

Принят протокол на основе **Double Ratchet Algorithm** (Signal Protocol) с модификациями для multi-device.

### Криптографические примитивы

| Компонент | Алгоритм | Обоснование |
|-----------|----------|-------------|
| Key Exchange | X25519 (ECDH) | Быстрый, безопасный, малый размер ключа |
| Symmetric Encryption | AES-256-GCM | Аппаратное ускорение, стандарт индустрии |
| Alternative Encryption | ChaCha20-Poly1305 | Для устройств без AES-NI |
| Hash Function | SHA-256 / BLAKE2b | Проверенные, быстрые |
| KDF | HKDF-SHA256 | Стандарт для key derivation |
| Signature | Ed25519 | Для верификации identity keys |

### Архитектура ключей

```
┌─────────────────────────────────────────────────────┐
│                 Identity Keys (Long-term)           │
│  - Generated once at install                        │
│  - Stored in Secure Enclave/Keystore                │
│  - Used for authentication only                     │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                Signed PreKeys (Medium-term)         │
│  - Rotated weekly                                   │
│  - Signed by Identity Key                           │
│  - Uploaded to server                               │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                 One-Time PreKeys (Ephemeral)        │
│  - Generated in batches (100)                       │
│  - Consumed on first use                            │
│  - Ensure forward secrecy                           │
└─────────────────────────────────────────────────────┘
```

### Double Ratchet механизм

```
Сессионные ключи эволюционируют двумя механизмами:

1. Symmetric-key ratchet (KDF chain)
   ┌─────────┐     ┌─────────┐     ┌─────────┐
   │ CK_prev │ ──► │  KDF    │ ──► │ CK_next │
   └─────────┘     └─────────┘     └─────────┘
                      │
                      ▼
                 ┌─────────┐
                 │Msg Key  │ → Шифрование сообщения
                 └─────────┘

2. Diffie-Hellman ratchet (асинхронный)
   При получении нового public key от собеседника:
   - Новый DH output
   - Сброс KDF chain
   - Post-compromise security
```

### Протокол установки сессии (X3DH)

```
Alice хочет начать чат с Bob:

1. Alice получает с сервера:
   - Identity Key Bob (IK_B)
   - Signed PreKey Bob (SPK_B)
   - One-Time PreKey Bob (OPK_B, опционально)

2. Alice генерирует:
   - Ephemeral Key (EK_A)
   - Вычисляет DH共享ты:
     * DH1 = DH(IK_A, SPK_B)
     * DH2 = DH(EK_A, IK_B)
     * DH3 = DH(EK_A, SPK_B)
     * DH4 = DH(EK_A, OPK_B) // если есть

3. SK = KDF(DH1 || DH2 || DH3 || DH4)

4. Alice отправляет Bob:
   - IK_A
   - EK_A
   - ID использованного OPK_B

5. Bob вычисляет тот же SK и начинает ratchet
```

### Multi-device синхронизация

```
Каждое устройство имеет независимые identity keys:

Device A1 ──┐
            ├──► Session with Bob's Device B1
Device A2 ──┤
            └──► Session with Bob's Device B2

Решения:
1. Sender Keys для групп
2. Sesion-per-device для 1:1
3. Server-assisted key distribution (без доступа к контенту)
```

## Формат зашифрованного сообщения

```go
type EncryptedMessage struct {
    Version      uint8    // Протокол version
    Type         uint8    // Тип: PREKEY_MESSAGE | NORMAL_MESSAGE
    
    // Header (нешифрованный)
    SenderKeyID  string   // ID отправителя
    ReceiverKeyID string  // ID получателя
    MessageCounter uint64 // Ratchet counter
    Timestamp    int64
    
    // Encrypted payload
    Ciphertext   []byte   // AES-GCM или ChaCha20-Poly1305
    AuthTag      []byte   // Authentication tag
    
    // Metadata (опционально шифруется)
    EncryptedMetadata []byte // Тип сообщения, attachments info
}
```

## Хранение ключей

### iOS
```swift
// Secure Enclave для Identity Keys
let attributes: [CFString: Any] = [
    kSecAttrAccessible: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
    kSecAttrTokenID: kSecAttrTokenIDSecureEnclave,
    kSecPrivateKeyAttrs: [
        kSecAttrIsPermanent: true,
        kSecAttrApplicationTag: "messenger.identity"
    ]
]
```

### Android
```kotlin
// StrongBox или Keystore
val params = KeyGenParameterSpec.Builder(
    "messenger_identity",
    KeyProperties.PURPOSE_SIGN or KeyProperties.PURPOSE_AGREE_KEY
)
    .setUserAuthenticationRequired(false)
    .setIsStrongBoxBacked(true) // Если доступен
    .build()
```

### Desktop
```rust
// OS keyring + master password
use keyring::Entry;

let entry = Entry::new("messenger", "identity_keys");
entry.set_password(&encrypted_keys)?;
```

## Threat Model

### Атаки и защита

| Атака | Защита |
|-------|--------|
| MITM при установке сессии | Fingerprint verification (QR/SAS) |
| Replay attack | Message counters, idempotency |
| Key compromise impersonation | Signed prekeys, identity verification |
| Traffic analysis | Metadata encryption, padding |
| Side-channel | Constant-time algorithms, blinding |
| Quantum computing (future) | Постквантовая миграция plan |

### Доверие серверу

```
Сервер НЕ знает:
❌ Content сообщений
❌ Session keys
❌ Identity private keys

Сервер ВИДИТ:
✅ Public keys (но не может расшифровать)
✅ Metadata (кто, когда, размер)
✅ Online status
✅ Device information
```

## Верификация ключей

### Safety Number (Signal-style)
```
SHA-256(IK_A || IK_B) → 60 digits → QR code + manual compare

Users verify:
1. In-person scan QR
2. Voice call read numbers
3. Secondary channel comparison
```

### Key Transparency Log
```
Все public keys записываются в append-only log:
- Clients audit log periodically
- Detect unauthorized key changes
- Similar to Certificate Transparency
```

## Последствия

### Положительные
- ✅ Industry-standard security (Signal-proven)
- ✅ Forward & backward secrecy
- ✅ Deniable authentication
- ✅ Post-compromise security

### Отрицательные
- ⚠️ Сложность реализации (криптография опасна)
- ⚠️ Performance overhead (DH operations)
- ⚠️ UX friction (key verification)
- ⚠️ Multi-device complexity

### Нейтральные
- ➖ Увеличенный размер сообщений (~200 bytes overhead)
- ➶ Необходимость backup/recovery механизма

## Альтернативы

### 1. TLS только (server-side encryption)
**Отклонено:** Сервер видит всё, нет доверия

### 2. PGP/GPG
**Отклонено:** Плохой UX, нет forward secrecy, no deniability

### 3. MLS (Messaging Layer Security)
**Отложено:** IETF стандарт, хорош для групп, но сложен для 1:1

## План внедрения

1. **Неделя 1-2:** Crypto library selection & audit
2. **Неделя 3-4:** Key generation & storage implementation
3. **Неделя 5-6:** X3DH protocol implementation
4. **Неделя 7-8:** Double Ratchet implementation
5. **Неделя 9-10:** Integration tests & test vectors
6. **Неделя 11-12:** Independent security audit
7. **Неделя 13-14:** Bug bounty program launch

## Метрики успеха

| Метрика | Target | Измерение |
|---------|--------|-----------|
| Key generation time | <500мс | On mid-range device |
| Message encryption | <10мс | Per message |
| Session establishment | <2 сек | Including network |
| Audit findings | 0 Critical | From independent auditor |
| False positive rate | 0% | Key verification |

## Ссылки

- [Signal Protocol Documentation](https://signal.org/docs/)
- [Double Ratchet Algorithm](https://www.signal.org/docs/specifications/doubleratchet/)
- [X3DH Protocol](https://www.signal.org/docs/specifications/x3dh/)
- [NIST SP 800-56A](https://csrc.nist.gov/publications/detail/sp/800-56a/rev-2/final)
- [RFC 7748 (Elliptic Curves)](https://datatracker.ietf.org/doc/html/rfc7748)

---

*ADR требует обновления после security audit*
