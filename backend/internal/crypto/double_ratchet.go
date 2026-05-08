// Double Ratchet implementation for forward secrecy and deniable authentication
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

// DHFunction represents a Diffie-Hellman function
type DHFunction func(private, public *[32]byte) ([32]byte, error)

// RatchetKeyPair represents keys for the ratchet
type RatchetKeyPair struct {
	Public  [32]byte
	Private [32]byte
}

// ChainKey represents a key in the symmetric key ratchet
type ChainKey struct {
	Key  []byte
	Counter uint32
}

// MessageKey represents a key used to encrypt a single message
type MessageKey struct {
	Key []byte
	Nonce []byte
}

// DoubleRatchet implements the Signal Protocol's Double Ratchet algorithm
type DoubleRatchet struct {
	// DH ratchet state
	dhSelf     *RatchetKeyPair
	dhRemote   [32]byte
	lastDHMessage bool

	// Symmetric key ratchet state
	rootKey      []byte
	sendingChain *ChainKey
	receivingChains map[string]*ChainKey // chainKey indexed by DH public key

	// Nkeys (number of messages sent/received with current chain)
	nSend uint32
	nRecv uint32

	// PN (previous N) for skipping message keys
	pn uint32

	dhFunc DHFunction
	kdf    func([]byte, []byte) ([]byte, []byte)
}

// NewDoubleRatchet creates a new Double Ratchet instance
func NewDoubleRatchet(dhSelf *RatchetKeyPair, dhRemote [32]byte, rootKey []byte, dhFunc DHFunction) *DoubleRatchet {
	return &DoubleRatchet{
		dhSelf:          dhSelf,
		dhRemote:        dhRemote,
		lastDHMessage:   false,
		rootKey:         rootKey,
		sendingChain:    nil,
		receivingChains: make(map[string]*ChainKey),
		nSend:           0,
		nRecv:           0,
		pn:              0,
		dhFunc:          dhFunc,
		kdf:             hkdfSHA256,
	}
}

// Encrypt encrypts a plaintext message
func (r *DoubleRatchet) Encrypt(plaintext []byte, associatedData []byte) ([]byte, []byte, error) {
	if r.sendingChain == nil {
		return nil, nil, errors.New("no sending chain available")
	}

	// Get message key
	mk := r.nextSendingMessageKey()

	// Encrypt using AES-GCM or ChaCha20-Poly1305
	ciphertext, _, err := ChaCha20Poly1305Encrypt(mk.Key, plaintext, associatedData)
	if err != nil {
		return nil, nil, err
	}

	r.nSend++
	r.lastDHMessage = false

	return ciphertext, mk.Nonce, nil
}

// Decrypt decrypts a ciphertext message
func (r *DoubleRatchet) Decrypt(ciphertext, nonce, associatedData, remoteEphemeral []byte) ([]byte, error) {
	if len(remoteEphemeral) > 0 {
		// DH ratchet step
		if err := r.dHRatchet(remoteEphemeral); err != nil {
			return nil, err
		}
	}

	// Try to find the message key in skipped message keys or current receiving chain
	var mk *MessageKey
	var found bool

	// Check skipped message keys first
	mk, found = r.getSkippedMessageKey(remoteEphemeral)
	if !found {
		// Try current receiving chain
		chainKey, exists := r.receivingChains[string(remoteEphemeral)]
		if !exists {
			return nil, errors.New("message key not found")
		}

		// Advance chain to get message key
		for chainKey.Counter < r.nRecv {
			chainKey = r.nextReceivingMessageKey(chainKey)
		}

		mk = &MessageKey{
			Key:   chainKey.Key,
			Nonce: make([]byte, 12),
		}
		copy(mk.Nonce, nonce)

		r.nRecv++
	}

	// Decrypt
	plaintext, err := ChaCha20Poly1305Decrypt(mk.Key, ciphertext, nonce, associatedData)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// dhRatchet performs a Diffie-Hellman ratchet step
func (r *DoubleRatchet) dHRatchet(remoteEphemeral []byte) error {
	var remotePub [32]byte
	copy(remotePub[:], remoteEphemeral)

	// DH output
	dhOut, err := r.dhFunc(&r.dhSelf.Private, &remotePub)
	if err != nil {
		return err
	}

	// Root key ratchet
	newRootKey, chainKey := r.kdf(r.rootKey, dhOut[:])
	r.rootKey = newRootKey

	// Update receiving chain
	r.receivingChains[string(r.dhSelf.Public[:])] = &ChainKey{
		Key:     chainKey,
		Counter: 0,
	}

	// Store previous N
	r.pn = r.nSend
	r.nSend = 0
	r.nRecv = 0

	// Generate new DH keys
	newDH, err := GenerateKeyPair()
	if err != nil {
		return err
	}

	r.dhSelf = &RatchetKeyPair{
		Public:  newDH.PublicKey,
		Private: newDH.PrivateKey,
	}

	// DH output for sending chain
	dhOut2, err := r.dhFunc(&r.dhSelf.Private, &r.dhRemote)
	if err != nil {
		return err
	}

	// Root key ratchet again
	newRootKey2, sendingChainKey := r.kdf(r.rootKey, dhOut2[:])
	r.rootKey = newRootKey2

	r.sendingChain = &ChainKey{
		Key:     sendingChainKey,
		Counter: 0,
	}

	r.lastDHMessage = true
	return nil
}

// nextSendingMessageKey generates the next message key for sending
func (r *DoubleRatchet) nextSendingMessageKey() *MessageKey {
	if r.sendingChain == nil {
		return nil
	}

	// HMAC-based key derivation
	h := hmac.New(sha256.New, r.sendingChain.Key)
	h.Write([]byte{0x01})
	messageKey := h.Sum(nil)

	// Update chain key
	h2 := hmac.New(sha256.New, r.sendingChain.Key)
	h2.Write([]byte{0x02})
	r.sendingChain.Key = h2.Sum(nil)
	r.sendingChain.Counter++

	// Generate nonce from message key
	h3 := hmac.New(sha256.New, messageKey)
	h3.Write([]byte{0x03})
	nonce := h3.Sum(nil)[:12]

	return &MessageKey{
		Key:   messageKey,
		Nonce: nonce,
	}
}

// nextReceivingMessageKey advances the receiving chain
func (r *DoubleRatchet) nextReceivingMessageKey(chainKey *ChainKey) *ChainKey {
	h := hmac.New(sha256.New, chainKey.Key)
	h.Write([]byte{0x01})

	h2 := hmac.New(sha256.New, chainKey.Key)
	h2.Write([]byte{0x02})
	chainKey.Key = h2.Sum(nil)
	chainKey.Counter++

	return chainKey
}

// getSkippedMessageKey tries to find a skipped message key
func (r *DoubleRatchet) getSkippedMessageKey(remoteEphemeral []byte) (*MessageKey, bool) {
	// Simplified implementation - in production, maintain a list of skipped keys
	return nil, false
}

// hkdfSHA256 implements HKDF-SHA256
func hkdfSHA256(key, data []byte) ([]byte, []byte) {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	prk := h.Sum(nil)

	h2 := hmac.New(sha256.New, prk)
	h2.Write([]byte{0x01})
	okm1 := h2.Sum(nil)

	h3 := hmac.New(sha256.New, prk)
	h3.Write(okm1)
	h3.Write([]byte{0x02})
	okm2 := h3.Sum(nil)

	return okm1, okm2
}

// X3DH implements the Extended Triple Diffie-Hellman key agreement
type X3DHState struct {
	identityKey    *KeyPair
	signedPreKey   *KeyPair
	oneTimePreKeys []*KeyPair
}

// NewX3DHState creates a new X3DH state
func NewX3DHState() (*X3DHState, error) {
	identityKey, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	signedPreKey, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	oneTimePreKeys := make([]*KeyPair, 10)
	for i := range oneTimePreKeys {
		key, err := GenerateKeyPair()
		if err != nil {
			return nil, err
		}
		oneTimePreKeys[i] = key
	}

	return &X3DHState{
		identityKey:    identityKey,
		signedPreKey:   signedPreKey,
		oneTimePreKeys: oneTimePreKeys,
	}, nil
}

// CalculateSharedSecret calculates the shared secret using X3DH
func CalculateSharedSecret(
	privateIdentity, privateSigned, privateOneTime *[32]byte,
	publicIdentity, publicSigned, publicOneTime *[32]byte,
) ([]byte, error) {
	var sharedSecret []byte

	// DH1: identity_key_A * signed_prekey_B
	dh1, err := ECDH(privateIdentity, publicSigned)
	if err != nil {
		return nil, err
	}
	sharedSecret = append(sharedSecret, dh1[:]...)

	// DH2: ephemeral_key_A * identity_key_B
	dh2, err := ECDH(privateSigned, publicIdentity)
	if err != nil {
		return nil, err
	}
	sharedSecret = append(sharedSecret, dh2[:]...)

	// DH3: ephemeral_key_A * signed_prekey_B
	dh3, err := ECDH(privateSigned, publicSigned)
	if err != nil {
		return nil, err
	}
	sharedSecret = append(sharedSecret, dh3[:]...)

	// DH4: ephemeral_key_A * one_time_prekey_B (if available)
	if privateOneTime != nil && publicOneTime != nil {
		dh4, err := ECDH(privateOneTime, publicOneTime)
		if err != nil {
			return nil, err
		}
		sharedSecret = append(sharedSecret, dh4[:]...)
	}

	// Hash the concatenated DH outputs
	hash := sha256.Sum256(sharedSecret)
	return hash[:], nil
}
