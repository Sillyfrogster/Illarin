package dispatch

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestAMintedSecretIsMarkedAndFullLength(t *testing.T) {
	t.Parallel()
	secret, err := MintSecret()

	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if !strings.HasPrefix(secret, Prefix) {
		t.Errorf("secret = %q, want the %s mark", secret, Prefix)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, Prefix))
	if err != nil || len(key) != SecretBytes {
		t.Errorf("secret holds %d bytes (%v), want %d", len(key), err, SecretBytes)
	}
}

func TestTwoMintedSecretsDiffer(t *testing.T) {
	t.Parallel()
	first, _ := MintSecret()
	second, _ := MintSecret()

	if first == second {
		t.Error("two minted secrets were the same")
	}
}

func TestTheSignatureIsOverIdTimestampAndBody(t *testing.T) {
	t.Parallel()
	key := make([]byte, SecretBytes)
	secret := Prefix + base64.StdEncoding.EncodeToString(key)
	at := time.Unix(1700000000, 0)
	body := []byte(`{"type":"publication.post.published.v1"}`)

	signature, err := Sign(secret, "msg_1", at, body)

	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("msg_1.1700000000." + string(body)))
	want := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if signature != want {
		t.Errorf("signature = %q, want %q", signature, want)
	}
}

func TestAChangedBodyChangesTheSignature(t *testing.T) {
	t.Parallel()
	secret, _ := MintSecret()
	at := time.Unix(1700000000, 0)

	first, _ := Sign(secret, "msg_1", at, []byte(`{"a":1}`))
	second, _ := Sign(secret, "msg_1", at, []byte(`{"a":2}`))

	if first == second {
		t.Error("two different bodies signed the same")
	}
}

func TestHeadersCarryTheIdTimestampAndSignature(t *testing.T) {
	t.Parallel()
	secret, _ := MintSecret()
	at := time.Unix(1700000000, 0)

	headers, err := Headers([]string{secret}, "msg_1", at, []byte(`{}`))

	if err != nil {
		t.Fatalf("headers: %v", err)
	}
	if headers[IDHeader] != "msg_1" {
		t.Errorf("%s = %q", IDHeader, headers[IDHeader])
	}
	if headers[TimestampHeader] != "1700000000" {
		t.Errorf("%s = %q", TimestampHeader, headers[TimestampHeader])
	}
	signature, _ := Sign(secret, "msg_1", at, []byte(`{}`))
	if headers[SignatureHeader] != signature {
		t.Errorf("%s = %q, want %q", SignatureHeader, headers[SignatureHeader], signature)
	}
}

func TestAValueThatIsNotASigningSecretIsRefused(t *testing.T) {
	t.Parallel()
	for _, secret := range []string{"", "abc", "whsec_not-base64!!", "whsec_" + base64.StdEncoding.EncodeToString([]byte("short"))} {
		if _, err := Sign(secret, "msg_1", time.Unix(0, 0), nil); err == nil {
			t.Errorf("Sign(%q) was accepted", secret)
		}
	}
}
