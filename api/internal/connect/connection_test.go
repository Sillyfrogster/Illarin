package connect

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCanonicalPermissionsAcceptsEachKnownPermissionOnce(t *testing.T) {
	t.Parallel()
	both, err := canonicalPermissions([]Permission{PermissionSyncLibrary, PermissionReceiveWorks})
	if err != nil {
		t.Fatalf("both permissions: %v", err)
	}
	if len(both) != 2 || both[0] != PermissionReceiveWorks || both[1] != PermissionSyncLibrary {
		t.Errorf("canonical order = %v", both)
	}

	one, err := canonicalPermissions([]Permission{PermissionSyncLibrary})
	if err != nil || len(one) != 1 || one[0] != PermissionSyncLibrary {
		t.Errorf("one permission = %v, %v", one, err)
	}

	if none, err := canonicalPermissions(nil); err != nil || len(none) != 0 {
		t.Errorf("no permissions = %v, %v, want an empty set", none, err)
	}

	for _, requested := range [][]Permission{
		{"asset:write"},
		{PermissionReceiveWorks, PermissionReceiveWorks},
		{PermissionReceiveWorks, "asset:write"},
	} {
		if _, err := canonicalPermissions(requested); !errors.Is(err, ErrInvalidPermissions) {
			t.Errorf("canonicalPermissions(%v) error = %v, want a refusal", requested, err)
		}
	}
}

func TestCapabilitiesAreNamespacedBoundedAndOrdered(t *testing.T) {
	t.Parallel()
	capabilities, err := validateCapabilities(testCapabilities())
	if err != nil {
		t.Fatalf("valid capabilities: %v", err)
	}
	if strings.Join(capabilities.Declared, ",") != "paper-lantern:install,paper-lantern:sync" {
		t.Errorf("capability order = %v", capabilities.Declared)
	}
	if strings.Join(capabilities.AcceptedFormats, ",") != "character-card-v3,lorebook-v2" {
		t.Errorf("format order = %v", capabilities.AcceptedFormats)
	}

	invalid := []Capabilities{
		{ProtocolVersion: 1},
		withDeclared(testCapabilities(), []string{"not-namespaced"}),
		withDeclared(testCapabilities(), []string{"paper-lantern:sync", "paper-lantern:sync"}),
		withFormats(testCapabilities(), []string{"UPPERCASE"}),
		withProtocol(testCapabilities(), 2),
	}
	for _, candidate := range invalid {
		if _, err := validateCapabilities(candidate); !errors.Is(err, ErrInvalidCapabilities) {
			t.Errorf("validateCapabilities(%+v) error = %v, want a refusal", candidate, err)
		}
	}
}

func TestAUserCodeIsReadTheWayACreatorTypesIt(t *testing.T) {
	t.Parallel()
	code, err := newCode(codeLength)
	if err != nil {
		t.Fatalf("new code: %v", err)
	}
	shown := FormatUserCode(code)

	for _, typed := range []string{shown, strings.ToLower(shown), code, "  " + shown + "  "} {
		normalized, ok := normalizeUserCode(typed)
		if !ok || normalized != code {
			t.Errorf("normalizeUserCode(%q) = %q, %v, want %q", typed, normalized, ok, code)
		}
	}
	for _, typed := range []string{
		"", "SHORT", code + "B", "AEIO" + code[4:], strings.Repeat("-", 1000) + code,
	} {
		if _, ok := normalizeUserCode(typed); ok {
			t.Errorf("normalizeUserCode(%q) accepted a code it cannot be", typed)
		}
	}
}

func TestOpaqueInputsAreRejectedBeforeDecodingUnboundedText(t *testing.T) {
	t.Parallel()
	code, _, err := newOpaqueCode()
	if err != nil {
		t.Fatalf("new opaque code: %v", err)
	}
	if _, ok := opaqueCodeHash(code); !ok {
		t.Fatal("a fresh opaque code was refused")
	}
	if _, ok := opaqueCodeHash(code + strings.Repeat("A", 1000)); ok {
		t.Fatal("an oversized opaque code was accepted")
	}

	token, _, _, err := newCredential(refreshTokenType)
	if err != nil {
		t.Fatalf("new credential: %v", err)
	}
	if _, ok := credentialHash(token+strings.Repeat("A", 1000), refreshTokenType); ok {
		t.Fatal("an oversized credential was accepted")
	}
}

func TestCredentialsHaveSeparateTypesAndRejectMalformedValues(t *testing.T) {
	t.Parallel()
	for _, secretType := range []string{accessTokenType, refreshTokenType} {
		token, prefix, hash, err := newCredential(secretType)
		if err != nil {
			t.Fatalf("new %s credential: %v", secretType, err)
		}
		if !strings.HasPrefix(token, secretType+"."+prefix+".") {
			t.Errorf("token %q does not carry type and prefix", token)
		}
		got, ok := credentialHash(token, secretType)
		if !ok || string(got) != string(hash) {
			t.Error("a fresh credential does not hash to what was stored")
		}
		other := accessTokenType
		if secretType == accessTokenType {
			other = refreshTokenType
		}
		if _, ok := credentialHash(token, other); ok {
			t.Errorf("%s credential was accepted as %s", secretType, other)
		}
	}

	for _, malformed := range []string{"", "no-dot", "ia1.SHORT.abc", "ia1.BAD-CODE.not-base64"} {
		if _, ok := credentialHash(malformed, accessTokenType); ok {
			t.Errorf("credentialHash(%q) accepted a malformed value", malformed)
		}
	}
}

func TestAuthorizationAcceptsOnlyExactLoopbackCallbacksAndS256(t *testing.T) {
	t.Parallel()
	verifier := strings.Repeat("v", 64)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	base := AuthorizationInput{
		StartInput:          testStart(PermissionReceiveWorks),
		RedirectURI:         "http://127.0.0.1:49152/link/callback",
		State:               strings.Repeat("s", 43),
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	}
	if _, err := validateAuthorization(base); err != nil {
		t.Fatalf("valid authorization: %v", err)
	}
	if !challengeMatches(verifier, challenge) || challengeMatches(strings.Repeat("x", 64), challenge) {
		t.Error("S256 verifier comparison did not bind the original verifier")
	}

	for _, callback := range []string{
		"http://[::1]:49152/link/callback",
		"http://127.0.0.1:1/link/callback",
	} {
		candidate := base
		candidate.RedirectURI = callback
		if _, err := validateAuthorization(candidate); err != nil {
			t.Errorf("loopback callback %q: %v", callback, err)
		}
	}
	for _, callback := range []string{
		"http://localhost:49152/link/callback",
		"http://127.0.0.2:49152/link/callback",
		"http://[::ffff:127.0.0.1]:49152/link/callback",
		"http://[0:0:0:0:0:0:0:1]:49152/link/callback",
		"http://192.168.1.4:49152/link/callback",
		"https://127.0.0.1:49152/link/callback",
		"http://127.0.0.1/link/callback",
		"http://127.0.0.1:49152/link/callback?code=old",
		"http://user@127.0.0.1:49152/link/callback",
		"http://127.0.0.1:49152/" + strings.Repeat("x", maxRedirectLength),
	} {
		candidate := base
		candidate.RedirectURI = callback
		if _, err := validateAuthorization(candidate); !errors.Is(err, ErrInvalidRedirect) {
			t.Errorf("callback %q error = %v, want refusal", callback, err)
		}
	}
}

func TestAConnectedAppIsRefusedAPermissionItWasNotGranted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	service := NewApps(
		pool,
		"http://localhost:3000",
		[]byte("01234567890123456789012345678901"),
	)
	creator := insertCreator(t, pool)

	started, err := service.Start(ctx, "127.0.0.1", testStart(PermissionReceiveWorks))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pending, err := service.Pending(ctx, creator, started.UserCode)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if _, err := service.Approve(ctx, creator, started.UserCode, pending.ApprovalToken, []Permission{PermissionReceiveWorks}); err != nil {
		t.Fatalf("approve: %v", err)
	}
	credentials, connected, err := service.Poll(ctx, "127.0.0.1", started.DeviceCode)
	if err != nil || !connected {
		t.Fatalf("poll: %v, connected %v", err, connected)
	}

	if _, err := service.Authenticate(ctx, credentials.AccessToken, PermissionReceiveWorks); err != nil {
		t.Errorf("granted permission refused: %v", err)
	}
	if _, err := service.Authenticate(ctx, credentials.AccessToken, PermissionSyncLibrary); !errors.Is(
		err, ErrMissingPermission,
	) {
		t.Errorf("ungranted permission error = %v, want a refusal", err)
	}
	if _, err := service.Authenticate(ctx, credentials.AccessToken, ""); err != nil {
		t.Errorf("an endpoint needing no permission refused a live credential: %v", err)
	}
}

func testCapabilities() Capabilities {
	return Capabilities{
		AppVersion: "2.4.0", ProtocolVersion: 1,
		Declared:        []string{"paper-lantern:install", "paper-lantern:sync"},
		AcceptedFormats: []string{"character-card-v3", "lorebook-v2"},
	}
}

func testStart(permissions ...Permission) StartInput {
	return StartInput{
		AppName: "Paper Lantern", Name: "studio workstation",
		Capabilities: testCapabilities(), Permissions: permissions,
	}
}

func withDeclared(capabilities Capabilities, declared []string) Capabilities {
	capabilities.Declared = declared
	return capabilities
}

func withFormats(capabilities Capabilities, formats []string) Capabilities {
	capabilities.AcceptedFormats = formats
	return capabilities
}

func withProtocol(capabilities Capabilities, version int) Capabilities {
	capabilities.ProtocolVersion = version
	return capabilities
}

func insertCreator(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username, email, email_source, email_verified_at)
		 values ($1, 'connect.creator', 'creator@example.com', 'creator', now())`,
		id); err != nil {
		t.Fatalf("create creator: %v", err)
	}
	return id
}
