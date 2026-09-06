// Package secrets seals the configuration Illarin has to be able to read back.
// A bearer credential is hashed and never recovered; a webhook signing secret
// and the address it is sent to have to survive a restart in a form the
// service can still use, so they are encrypted under one operator-held key.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// KeyBytes is the length of the key an operator supplies.
const KeyBytes = 32

// ErrUnreadable says the stored bytes were not written by this key.
var ErrUnreadable = errors.New("the sealed value cannot be read with this key")

// Key seals and opens one deployment's recoverable secrets.
type Key struct {
	sealer cipher.AEAD
}

// NewKey takes the operator's key and refuses anything but the exact length.
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

// Ready answers whether this key was configured at all, so a service holding
// the zero value refuses the work rather than writing something it cannot read.
func (k Key) Ready() bool { return k.sealer != nil }

// Seal returns bytes only this key can turn back into the value.
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

// Open returns the value behind bytes this key sealed.
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
