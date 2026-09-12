package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

const KeyBytes = 32

var ErrUnreadable = errors.New("the sealed value cannot be read with this key")

type Key struct {
	sealer cipher.AEAD
}

func NewKey(raw []byte) (Key, error) {
	if len(raw) != KeyBytes {
		return Key{}, fmt.Errorf("a sealing key is %d bytes, got %d", KeyBytes, len(raw))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return Key{}, fmt.Errorf("read the sealing key: %w", err)
	}
	sealer, err := cipher.NewGCM(block)
	if err != nil {
		return Key{}, fmt.Errorf("read the sealing key: %w", err)
	}
	return Key{sealer: sealer}, nil
}

func (k Key) Ready() bool { return k.sealer != nil }

func (k Key) Seal(plain []byte) ([]byte, error) {
	if !k.Ready() {
		return nil, errors.New("no sealing key is configured")
	}
	nonce := make([]byte, k.sealer.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("seal a value: %w", err)
	}
	return k.sealer.Seal(nonce, nonce, plain, nil), nil
}

func (k Key) Open(sealed []byte) ([]byte, error) {
	if !k.Ready() {
		return nil, errors.New("no sealing key is configured")
	}
	size := k.sealer.NonceSize()
	if len(sealed) < size {
		return nil, ErrUnreadable
	}
	plain, err := k.sealer.Open(nil, sealed[:size], sealed[size:], nil)
	if err != nil {
		return nil, ErrUnreadable
	}
	return plain, nil
}
