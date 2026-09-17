package dispatch

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const Prefix = "whsec_"

const (
	IDHeader        = "webhook-id"
	TimestampHeader = "webhook-timestamp"
	SignatureHeader = "webhook-signature"
)

const SecretBytes = 32

const Version = "v1"

func MintSecret() (string, error) {
	body := make([]byte, SecretBytes)
	if _, err := rand.Read(body); err != nil {
		return "", fmt.Errorf("make a signing secret: %w", err)
	}
	return Prefix + base64.StdEncoding.EncodeToString(body), nil
}

func Sign(secret, id string, at time.Time, body []byte) (string, error) {
	key, err := keyOf(secret)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	fmt.Fprintf(mac, "%s.%d.", id, at.Unix())
	mac.Write(body)
	return Version + "," + base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func Headers(secrets []string, id string, at time.Time, body []byte) (map[string]string, error) {
	if len(secrets) == 0 {
		return nil, errors.New("a signed request carries at least one secret")
	}
	signatures := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		signature, err := Sign(secret, id, at, body)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, signature)
	}
	return map[string]string{
		IDHeader:        id,
		TimestampHeader: strconv.FormatInt(at.Unix(), 10),
		SignatureHeader: strings.Join(signatures, " "),
	}, nil
}

func Accepts(header, secret, id string, at time.Time, body []byte) bool {
	want, err := Sign(secret, id, at, body)
	if err != nil {
		return false
	}
	for _, held := range strings.Fields(header) {
		if hmac.Equal([]byte(held), []byte(want)) {
			return true
		}
	}
	return false
}

func keyOf(secret string) ([]byte, error) {
	if !strings.HasPrefix(secret, Prefix) {
		return nil, errors.New("a signing secret starts with " + Prefix)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, Prefix))
	if err != nil || len(key) != SecretBytes {
		return nil, fmt.Errorf("a signing secret holds %d encoded bytes", SecretBytes)
	}
	return key, nil
}
