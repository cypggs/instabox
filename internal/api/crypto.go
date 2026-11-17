package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// cryptoBox provides AES-GCM encryption for at-rest secrets (env vars, secret values).
type cryptoBox struct {
	mu   sync.RWMutex
	aead cipher.AEAD
}

var box = &cryptoBox{}

func init() { _ = initCrypto(nil) }

// InitCrypto (re)initializes the box with a user-provided secret.
func (s *Server) InitCrypto(secret []byte) error { return initCrypto(secret) }

func initCrypto(secret []byte) error {
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return err
		}
	}
	if len(secret) < 32 {
		padded := make([]byte, 32)
		copy(padded, secret)
		secret = padded
	}
	block, err := aes.NewCipher(secret[:32])
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	box.mu.Lock()
	box.aead = aead
	box.mu.Unlock()
	return nil
}

func (s *Server) encryptJSON(v any) ([]byte, error) {
	box.mu.RLock()
	aead := box.aead
	box.mu.RUnlock()
	if aead == nil {
		return nil, errors.New("crypto not initialized")
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, b, nil), nil
}

func (s *Server) decryptJSON(blob []byte, out any) error {
	box.mu.RLock()
	aead := box.aead
	box.mu.RUnlock()
	if aead == nil {
		return errors.New("crypto not initialized")
	}
	ns := aead.NonceSize()
	if len(blob) < ns {
		return errors.New("ciphertext too short")
	}
	plain, err := aead.Open(nil, blob[:ns], blob[ns:], nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, out)
}
