package crypto

import (
"bytes"
"testing"
)

func TestGenerateKeyPair(t *testing.T) {
keyPair, err := GenerateKeyPair()
if err != nil {
t.Fatalf("Failed to generate key pair: %v", err)
}

if keyPair == nil {
t.Error("KeyPair should not be nil")
}

if len(keyPair.PrivateKey) != 32 {
t.Errorf("Expected private key length 32, got %d", len(keyPair.PrivateKey))
}

if len(keyPair.PublicKey) != 32 {
t.Errorf("Expected public key length 32, got %d", len(keyPair.PublicKey))
}
}

func TestECDH(t *testing.T) {
keyPairA, err := GenerateKeyPair()
if err != nil {
t.Fatalf("Failed to generate key pair A: %v", err)
}

keyPairB, err := GenerateKeyPair()
if err != nil {
t.Fatalf("Failed to generate key pair B: %v", err)
}

secretA, err := ECDH(&keyPairA.PrivateKey, &keyPairB.PublicKey)
if err != nil {
t.Fatalf("Failed to derive shared secret A: %v", err)
}

secretB, err := ECDH(&keyPairB.PrivateKey, &keyPairA.PublicKey)
if err != nil {
t.Fatalf("Failed to derive shared secret B: %v", err)
}

if secretA != secretB {
t.Error("Shared secrets should be identical")
}
}

func TestEncryptDecryptAES(t *testing.T) {
key := make([]byte, 32)
for i := 0; i < 32; i++ {
key[i] = byte(i)
}

plaintext := []byte("Hello, World! This is a secret message.")

ciphertext, nonce, err := AESGCMEncrypt(key, plaintext, nil)
if err != nil {
t.Fatalf("Failed to encrypt: %v", err)
}

if ciphertext == nil {
t.Error("Ciphertext should not be nil")
}

if nonce == nil {
t.Error("Nonce should not be nil")
}

decrypted, err := AESGCMDecrypt(key, ciphertext, nonce, nil)
if err != nil {
t.Fatalf("Failed to decrypt: %v", err)
}

if !bytes.Equal(plaintext, decrypted) {
t.Errorf("Decrypted text doesn't match original. Got: %s", string(decrypted))
}
}

func TestEncryptDecryptChaCha20(t *testing.T) {
key := make([]byte, 32)
for i := 0; i < 32; i++ {
key[i] = byte(i)
}

plaintext := []byte("Hello, World! ChaCha20 encryption test.")

ciphertext, nonce, err := ChaCha20Poly1305Encrypt(key, plaintext, nil)
if err != nil {
t.Fatalf("Failed to encrypt: %v", err)
}

if ciphertext == nil {
t.Error("Ciphertext should not be nil")
}

if nonce == nil {
t.Error("Nonce should not be nil")
}

decrypted, err := ChaCha20Poly1305Decrypt(key, ciphertext, nonce, nil)
if err != nil {
t.Fatalf("Failed to decrypt: %v", err)
}

if !bytes.Equal(plaintext, decrypted) {
t.Errorf("Decrypted text doesn't match original. Got: %s", string(decrypted))
}
}

func TestDifferentPlaintextsProduceDifferentCiphertexts(t *testing.T) {
key := make([]byte, 32)
for i := 0; i < 32; i++ {
key[i] = byte(i)
}

plaintext := []byte("Same message twice")

ciphertext1, nonce1, _ := AESGCMEncrypt(key, plaintext, nil)
ciphertext2, nonce2, _ := AESGCMEncrypt(key, plaintext, nil)

if bytes.Equal(ciphertext1, ciphertext2) {
t.Error("Same plaintext should produce different ciphertexts (due to random nonce)")
}

decrypted1, _ := AESGCMDecrypt(key, ciphertext1, nonce1, nil)
decrypted2, _ := AESGCMDecrypt(key, ciphertext2, nonce2, nil)

if !bytes.Equal(decrypted1, decrypted2) {
t.Error("Both ciphertexts should decrypt to the same plaintext")
}
}

func TestEmptyPlaintext(t *testing.T) {
key := make([]byte, 32)
for i := 0; i < 32; i++ {
key[i] = byte(i)
}

plaintext := []byte("")

ciphertext, nonce, err := AESGCMEncrypt(key, plaintext, nil)
if err != nil {
t.Fatalf("Failed to encrypt empty plaintext: %v", err)
}

decrypted, err := AESGCMDecrypt(key, ciphertext, nonce, nil)
if err != nil {
t.Fatalf("Failed to decrypt: %v", err)
}

if len(decrypted) != 0 {
t.Errorf("Expected empty decrypted text, got %d bytes", len(decrypted))
}
}

func TestInvalidKeySize(t *testing.T) {
key := make([]byte, 16)
plaintext := []byte("Test")

_, _, err := AESGCMEncrypt(key, plaintext, nil)
if err == nil {
t.Error("Expected error for invalid key size")
}
}

func TestTamperedCiphertext(t *testing.T) {
key := make([]byte, 32)
for i := 0; i < 32; i++ {
key[i] = byte(i)
}

plaintext := []byte("Original message")
ciphertext, nonce, _ := AESGCMEncrypt(key, plaintext, nil)

ciphertext[0] ^= 0xFF

_, err := AESGCMDecrypt(key, ciphertext, nonce, nil)
if err == nil {
t.Error("Expected error when decrypting tampered ciphertext")
}
}
