// Package webhook stamps one outbound request the way Standard Webhooks
// describes, so a receiver can check authenticity with a documented convention
// rather than one Illarin invented.
package webhook

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

// Prefix marks a value as a webhook signing secret rather than any other
// secret Illarin hands out.
const Prefix = "whsec_"

// The headers every signed request carries.
const (
	IDHeader        = "webhook-id"
	TimestampHeader = "webhook-timestamp"
	SignatureHeader = "webhook-signature"
)

// SecretBytes is the length of the key behind one endpoint's secret.
const SecretBytes = 32

// Version marks which signing scheme produced a signature.
const Version = "v1"

// MintSecret draws one endpoint's signing secret. It is shown once and cannot
// be recovered from anything but the sealed copy.
func MintSecret() (string, error) {
	body := make([]byte, SecretBytes)
	if _, err := rand.Read(body); err != nil {
		return "", fmt.Errorf("make a signing secret: %w", err)
	}
	return Prefix + base64.StdEncoding.EncodeToString(body), nil
}

// Sign answers the signature for the exact bytes that will be sent.
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

// Headers answers what one signed request sends alongside its body.
func Headers(secret, id string, at time.Time, body []byte) (map[string]string, error) {
	signature, err := Sign(secret, id, at, body)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		IDHeader:        id,
		TimestampHeader: strconv.FormatInt(at.Unix(), 10),
		SignatureHeader: signature,
	}, nil
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
