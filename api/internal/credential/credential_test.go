package credential

import (
	"strings"
	"testing"
)

func TestAFreshSecretReadsBackAsWhatWasStored(t *testing.T) {
	minted, err := Mint(Publication)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if !strings.HasPrefix(minted.Value, string(Publication)+"."+minted.Prefix+".") {
		t.Fatalf("secret %q does not carry its kind and prefix", minted.Value)
	}
	read, ok := Read(minted.Value, Publication)
	if !ok {
		t.Fatal("a fresh secret was refused")
	}
	if read.Prefix != minted.Prefix || string(read.Hash) != string(minted.Hash) {
		t.Fatalf("read back %+v, minted %+v", read, minted)
	}
}

func TestTwoMintsShareNothing(t *testing.T) {
	first, err := Mint(Publication)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	second, err := Mint(Publication)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if first.Value == second.Value || first.Prefix == second.Prefix {
		t.Fatal("two mints produced the same value")
	}
}

func TestOneKindNeverReadsAsAnother(t *testing.T) {
	minted, err := Mint(InstanceAccess)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	for _, other := range []Kind{InstanceRefresh, Publication} {
		if _, ok := Read(minted.Value, other); ok {
			t.Fatalf("an %s secret was accepted as %s", InstanceAccess, other)
		}
	}
}

func TestMalformedAndOversizedSecretsAreRefusedBeforeDecoding(t *testing.T) {
	minted, err := Mint(Publication)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	refused := []string{
		"",
		"no-dot",
		"ip1.SHORT.abc",
		"ip1.BAD-CODE.not-base64",
		minted.Value + strings.Repeat("A", 1000),
		minted.Value[:len(minted.Value)-1],
	}
	for _, value := range refused {
		if _, ok := Read(value, Publication); ok {
			t.Errorf("Read(%q) accepted a malformed value", value)
		}
	}
}

func TestMatchesComparesWholeDigests(t *testing.T) {
	first, err := Mint(Publication)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	second, err := Mint(Publication)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if !Matches(first.Hash, first.Hash) {
		t.Fatal("a digest did not match itself")
	}
	if Matches(first.Hash, second.Hash) || Matches(first.Hash, nil) {
		t.Fatal("two different digests matched")
	}
}
