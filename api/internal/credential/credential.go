package credential

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

type Kind string

const (
	InstanceAccess  Kind = "ia1"
	InstanceRefresh Kind = "ir1"
)

type Secret struct {
	Value  string
	Prefix string
	Hash   []byte
}

const (
	kindLength   = 3
	alphabet     = "BCDFGHJKLMNPQRSTVWXZ23456789"
	prefixLength = 8
	bodyBytes    = 32
	bodyLength   = 43
	valueLength  = kindLength + 1 + prefixLength + 1 + bodyLength
)

func Mint(kind Kind) (Secret, error) {
	prefix, err := newPrefix()
	if err != nil {
		return Secret{}, err
	}
	body := make([]byte, bodyBytes)
	if _, err := rand.Read(body); err != nil {
		return Secret{}, fmt.Errorf("make secret: %w", err)
	}
	value := string(kind) + "." + prefix + "." + base64.RawURLEncoding.EncodeToString(body)
	return Secret{Value: value, Prefix: prefix, Hash: hashOf(value)}, nil
}

func Read(value string, kind Kind) (Secret, bool) {
	if len(value) != valueLength {
		return Secret{}, false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 || parts[0] != string(kind) || len(parts[1]) != prefixLength {
		return Secret{}, false
	}
	for _, letter := range parts[1] {
		if !strings.ContainsRune(alphabet, letter) {
			return Secret{}, false
		}
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(body) != bodyBytes {
		return Secret{}, false
	}
	return Secret{Value: value, Prefix: parts[1], Hash: hashOf(value)}, true
}

func Matches(one, other []byte) bool {
	return subtle.ConstantTimeCompare(one, other) == 1
}

func newPrefix() (string, error) {
	limit := 256 / len(alphabet) * len(alphabet)
	prefix := make([]byte, 0, prefixLength)
	buffer := make([]byte, prefixLength)
	for len(prefix) < prefixLength {
		if _, err := rand.Read(buffer); err != nil {
			return "", fmt.Errorf("make prefix: %w", err)
		}
		for _, value := range buffer {
			if int(value) < limit && len(prefix) < prefixLength {
				prefix = append(prefix, alphabet[int(value)%len(alphabet)])
			}
		}
	}
	return string(prefix), nil
}

func hashOf(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
