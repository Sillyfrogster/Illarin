package secrets

import (
	"bytes"
	"testing"
)

func TestSealedValueReadsBackUnchanged(t *testing.T) {
	t.Parallel()
	key, err := NewKey(bytes.Repeat([]byte{7}, KeyBytes))
	if err != nil {
		t.Fatalf("new key: %v", err)
	}

	sealed, err := key.Seal([]byte("whsec_abc"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	opened, err := key.Open(sealed)

	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(opened) != "whsec_abc" {
		t.Errorf("opened = %q, want %q", opened, "whsec_abc")
	}
}

func TestSealedValueDoesNotHoldThePlainText(t *testing.T) {
	t.Parallel()
	key, _ := NewKey(bytes.Repeat([]byte{7}, KeyBytes))

	sealed, err := key.Seal([]byte("https://hooks.example.com/very-secret-path"))

	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Contains(sealed, []byte("very-secret-path")) {
		t.Error("the sealed bytes still contain the address")
	}
}

func TestSealingTheSameValueTwiceDiffers(t *testing.T) {
	t.Parallel()
	key, _ := NewKey(bytes.Repeat([]byte{7}, KeyBytes))

	first, _ := key.Seal([]byte("whsec_abc"))
	second, _ := key.Seal([]byte("whsec_abc"))

	if bytes.Equal(first, second) {
		t.Error("two sealings produced the same bytes")
	}
}

func TestAnotherKeyCannotOpenIt(t *testing.T) {
	t.Parallel()
	mine, _ := NewKey(bytes.Repeat([]byte{7}, KeyBytes))
	theirs, _ := NewKey(bytes.Repeat([]byte{9}, KeyBytes))
	sealed, _ := mine.Seal([]byte("whsec_abc"))

	_, err := theirs.Open(sealed)

	if err == nil {
		t.Error("a different key opened the value")
	}
}

func TestTamperedBytesAreRefused(t *testing.T) {
	t.Parallel()
	key, _ := NewKey(bytes.Repeat([]byte{7}, KeyBytes))
	sealed, _ := key.Seal([]byte("whsec_abc"))
	sealed[len(sealed)-1] ^= 0xff

	_, err := key.Open(sealed)

	if err == nil {
		t.Error("tampered bytes opened")
	}
}

func TestShortBytesAreRefused(t *testing.T) {
	t.Parallel()
	key, _ := NewKey(bytes.Repeat([]byte{7}, KeyBytes))

	_, err := key.Open([]byte{1, 2, 3})

	if err == nil {
		t.Error("a value too short to hold a nonce opened")
	}
}

func TestAKeyOfTheWrongLengthIsRefused(t *testing.T) {
	t.Parallel()
	for _, length := range []int{0, 16, 31, 33, 64} {
		if _, err := NewKey(bytes.Repeat([]byte{1}, length)); err == nil {
			t.Errorf("a %d-byte key was accepted", length)
		}
	}
}
