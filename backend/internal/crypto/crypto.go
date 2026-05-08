// Crypto package implements end-to-end encryption using Double Ratchet algorithm
// Based on Signal Protocol: X3DH key agreement + Double Ratchet for message encryption
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/chacha20poly1305"
)

// KeyPair represents a Curve25519 key pair
type KeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// GenerateKeyPair generates a new Curve25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	privateKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, privateKey); err != nil {
		return nil, err
	}

	var publicKey, privKeyArr [32]byte
	copy(privKeyArr[:], privateKey)
	curve25519.ScalarBaseMult(&publicKey, &privKeyArr)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privKeyArr,
	}, nil
}

// ECDH performs Elliptic Curve Diffie-Hellman key exchange
func ECDH(privateKey, publicKey *[32]byte) ([32]byte, error) {
	var sharedSecret [32]byte
	if _, err := curve25519.X25519(privateKey[:], publicKey[:]); err != nil {
		return sharedSecret, err
	}
	copy(sharedSecret[:], privateKey[:])
	return sharedSecret, nil
}

// AESGCM encrypts data using AES-256-GCM
func AESGCMEncrypt(key, plaintext, associatedData []byte) (ciphertext []byte, nonce []byte, err error) {
	if len(key) != 32 {
		return nil, nil, errors.New("key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = aesgcm.Seal(nil, nonce, plaintext, associatedData)
	return ciphertext, nonce, nil
}

// AESGCMDecrypt decrypts data using AES-256-GCM
func AESGCMDecrypt(key, ciphertext, nonce, associatedData []byte) (plaintext []byte, err error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err = aesgcm.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// ChaCha20Poly1305Encrypt encrypts data using ChaCha20-Poly1305
func ChaCha20Poly1305Encrypt(key, plaintext, associatedData []byte) (ciphertext []byte, nonce []byte, err error) {
	if len(key) != 32 {
		return nil, nil, errors.New("key must be 32 bytes")
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = aead.Seal(nil, nonce, plaintext, associatedData)
	return ciphertext, nonce, nil
}

// ChaCha20Poly1305Decrypt decrypts data using ChaCha20-Poly1305
func ChaCha20Poly1305Decrypt(key, ciphertext, nonce, associatedData []byte) (plaintext []byte, err error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	plaintext, err = aead.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncodeBase64 encodes bytes to base64 string
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 string to bytes
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
