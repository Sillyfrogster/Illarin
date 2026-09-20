package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const (
	linkingKey     = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	integrationKey = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
)

func setLinkingKey(t *testing.T) {
	t.Helper()
	t.Setenv("LINKING_HMAC_KEY", linkingKey)
	t.Setenv("INTEGRATION_SECRET_KEY", integrationKey)
}

func setBlogURL(t *testing.T) {
	t.Helper()
	t.Setenv("BLOG_URL", "https://blog.illarin.test")
}

func TestLoadRequiresTheBlogOrigin(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")

	for _, value := range []string{
		"",
		"blog.illarin.test",
		"ftp://blog.illarin.test",
		"https://blog.illarin.test/blog",
		"https://blog.illarin.test/?draft=1",
		"https://blog.illarin.test/#top",
		"https://user:secret@blog.illarin.test",
	} {
		t.Setenv("BLOG_URL", value)
		if _, err := Load(); err == nil {
			t.Errorf("Load accepted BLOG_URL %q", value)
		}
	}

	for _, value := range []string{"https://blog.illarin.test", "http://blog.localhost:8000/"} {
		t.Setenv("BLOG_URL", value)
		cfg, err := Load()
		if err != nil {
			t.Errorf("Load refused BLOG_URL %q: %v", value, err)
			continue
		}
		if cfg.BlogURL != value {
			t.Errorf("BlogURL = %q, want %q", cfg.BlogURL, value)
		}
	}
}

func TestLoadRequiresAnExactUnpaddedIntegrationSecretKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("LINKING_HMAC_KEY", linkingKey)
	t.Setenv("PUBLICATION_SECRET_KEY", "")

	for _, key := range []string{
		"",
		"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB==",
		"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
	} {
		t.Setenv("INTEGRATION_SECRET_KEY", key)
		if _, err := Load(); err == nil {
			t.Errorf("Load accepted publication secret key %q", key)
		}
	}

	t.Setenv("INTEGRATION_SECRET_KEY", integrationKey)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with a 32-byte key: %v", err)
	}
	if len(cfg.IntegrationSecretKey) != 32 {
		t.Errorf("IntegrationSecretKey is %d bytes, want 32", len(cfg.IntegrationSecretKey))
	}

	t.Setenv("INTEGRATION_SECRET_KEY", "")
	t.Setenv("PUBLICATION_SECRET_KEY", integrationKey)
	older, err := Load()
	if err != nil {
		t.Fatalf("Load with the name the key had before 18 November 2026: %v", err)
	}
	if len(older.IntegrationSecretKey) != 32 {
		t.Errorf("the older name gave %d bytes, want 32", len(older.IntegrationSecretKey))
	}
}

func TestTheIntegrationSecretKeyCannotBeTheLinkingKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("LINKING_HMAC_KEY", linkingKey)
	t.Setenv("INTEGRATION_SECRET_KEY", linkingKey)

	if _, err := Load(); err == nil {
		t.Error("Load accepted one key doing two jobs")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is missing")
	}
}

func TestLoadUsesDefaultPort(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
}

func TestLoadRequiresAnExactUnpaddedLinkingKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("INTEGRATION_SECRET_KEY", integrationKey)

	for _, key := range []string{
		"",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		t.Setenv("LINKING_HMAC_KEY", key)
		if _, err := Load(); err == nil {
			t.Errorf("Load accepted linking key %q", key)
		}
	}

	t.Setenv("LINKING_HMAC_KEY", linkingKey)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with a 32-byte key: %v", err)
	}
	if len(cfg.LinkingHMACKey) != 32 || !bytes.Equal(cfg.LinkingHMACKey, make([]byte, 32)) {
		t.Errorf("decoded linking key has %d bytes or the wrong value", len(cfg.LinkingHMACKey))
	}
}

func TestLoadRejectsAnUnreadableUploadCeiling(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("MAX_UPLOAD_BYTES", "55mb")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error: a ceiling nobody can read must not fall back to the default")
	}
}

func TestLoadRejectsIncompleteSMTPSettings(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("SMTP_ADDR", "smtp.example.com:587")
	t.Setenv("SMTP_FROM", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when an SMTP server has no sender address")
	}
}

func TestReleaseRequiresAnAccountEmailTransport(t *testing.T) {
	setLinkingKey(t)
	setBlogURL(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_test")
	t.Setenv("UPLOADS_DIR", t.TempDir())
	for _, name := range []string{
		"SMTP_ADDR", "SMTP_FROM", "SMTP_USERNAME", "SMTP_PASSWORD",
		"MICROSOFT_365_TENANT_ID", "MICROSOFT_365_CLIENT_ID",
		"MICROSOFT_365_MAILBOX", "MICROSOFT_365_CLIENT_SECRET_FILE",
	} {
		t.Setenv(name, "")
	}
	t.Setenv("GIN_MODE", "release")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted logging account email instead of sending it")
	}
	t.Setenv("SMTP_ADDR", "smtp.illarin.test:587")
	t.Setenv("SMTP_FROM", "mail@illarin.test")
	if _, err := Load(); err != nil {
		t.Fatalf("production with SMTP: %v", err)
	}
	t.Setenv("SMTP_ADDR", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("GIN_MODE", "debug")
	if _, err := Load(); err != nil {
		t.Fatalf("local development with logged email: %v", err)
	}
}

func TestLoadReadsMicrosoft365SecretFromAFile(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("MICROSOFT_365_TENANT_ID", "tenant")
	t.Setenv("MICROSOFT_365_CLIENT_ID", "client")
	t.Setenv("MICROSOFT_365_MAILBOX", "mail@illarin.test")
	secretFile := filepath.Join(t.TempDir(), "client-secret")
	if err := os.WriteFile(secretFile, []byte("secret\n"), 0o600); err != nil {
		t.Fatalf("write client secret: %v", err)
	}
	t.Setenv("MICROSOFT_365_CLIENT_SECRET_FILE", secretFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Microsoft365.ClientSecret != "secret" || cfg.Microsoft365.Mailbox != "mail@illarin.test" {
		t.Fatalf("Microsoft 365 settings = %+v", cfg.Microsoft365)
	}
}

func TestLoadRejectsIncompleteMicrosoft365Settings(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("MICROSOFT_365_TENANT_ID", "tenant")

	if _, err := Load(); err == nil {
		t.Fatal("expected incomplete Microsoft 365 settings to fail")
	}
}

func TestLoadUsesSettledUploadLimits(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	for _, name := range []string{
		"MAX_UPLOAD_BYTES",
		"MAX_ARCHIVE_ENTRIES",
		"MAX_ARCHIVE_ENTRY_BYTES",
		"MAX_ARCHIVE_BYTES",
		"MAX_ARCHIVE_COMPRESSION_RATIO",
		"UPLOAD_WORKERS",
		"STORAGE_FREE_SPACE_RESERVE_BYTES",
		"ACCOUNT_STORAGE_CAP_BYTES",
	} {
		t.Setenv(name, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MaxUploadBytes != 32<<20 {
		t.Errorf("upload limit = %d, want 32 MB", cfg.MaxUploadBytes)
	}
	if cfg.ProbeLimits.MaxArchiveEntries != 4096 {
		t.Errorf("archive entries = %d, want 4096", cfg.ProbeLimits.MaxArchiveEntries)
	}
	if cfg.ProbeLimits.MaxEntryBytes != 32<<20 {
		t.Errorf("entry limit = %d, want 32 MB", cfg.ProbeLimits.MaxEntryBytes)
	}
	if cfg.ProbeLimits.MaxArchiveBytes != 128<<20 {
		t.Errorf("archive limit = %d, want 128 MB", cfg.ProbeLimits.MaxArchiveBytes)
	}
	if cfg.ProbeLimits.MaxCompressionRatio != 100 {
		t.Errorf("compression ratio = %v, want 100", cfg.ProbeLimits.MaxCompressionRatio)
	}
	if cfg.UploadWorkers != 2 {
		t.Errorf("upload workers = %d, want 2", cfg.UploadWorkers)
	}
	if cfg.StorageFreeSpaceReserveBytes != 5<<30 {
		t.Errorf("storage reserve = %d, want 5 GB", cfg.StorageFreeSpaceReserveBytes)
	}
	if cfg.AccountStorageCapBytes != 1<<30 {
		t.Errorf("account storage cap = %d, want 1 GB", cfg.AccountStorageCapBytes)
	}
}

func TestLoadReadsUploadLimitOverrides(t *testing.T) {
	setLinkingKey(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/illarin_dev")
	t.Setenv("UPLOADS_DIR", "/tmp/uploads")
	setBlogURL(t)
	t.Setenv("MAX_UPLOAD_BYTES", "101")
	t.Setenv("MAX_ARCHIVE_ENTRIES", "102")
	t.Setenv("MAX_ARCHIVE_ENTRY_BYTES", "103")
	t.Setenv("MAX_ARCHIVE_BYTES", "104")
	t.Setenv("MAX_ARCHIVE_COMPRESSION_RATIO", "10.5")
	t.Setenv("UPLOAD_WORKERS", "3")
	t.Setenv("STORAGE_FREE_SPACE_RESERVE_BYTES", "105")
	t.Setenv("ACCOUNT_STORAGE_CAP_BYTES", "106")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MaxUploadBytes != 101 || cfg.ProbeLimits.MaxArchiveEntries != 102 ||
		cfg.ProbeLimits.MaxEntryBytes != 103 || cfg.ProbeLimits.MaxArchiveBytes != 104 ||
		cfg.ProbeLimits.MaxCompressionRatio != 10.5 || cfg.UploadWorkers != 3 ||
		cfg.StorageFreeSpaceReserveBytes != 105 || cfg.AccountStorageCapBytes != 106 {
		t.Fatalf("overridden upload settings = %+v", cfg)
	}
}
