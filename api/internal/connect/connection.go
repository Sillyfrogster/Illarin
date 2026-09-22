package connect

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Sillyfrogster/Illarin/api/internal/credential"
	"github.com/google/uuid"
)

var permissionOrder = []Permission{PermissionReceiveWorks, PermissionSyncLibrary}

var (
	ErrInvalidName         = errors.New("the app and the connected app need names")
	ErrInvalidCapabilities = errors.New("the capabilities are not valid")
	ErrInvalidPermissions  = errors.New("only known permissions, each at most once, are allowed")
	ErrInvalidRedirect     = errors.New("the redirect is not an exact loopback callback")
	ErrInvalidPKCE         = errors.New("S256 PKCE data is not valid")
	ErrRequestNotFound     = errors.New("no pending connection request has that code")
	ErrRequestExpired      = errors.New("the connection request expired")
	ErrAccessDenied        = errors.New("the creator denied the connection request")
	ErrPollTooSoon         = errors.New("the client must slow down")
	ErrTooManyCodes        = errors.New("too many connection codes were entered")
	ErrTooManyRequests     = errors.New("too many connection requests were made")
	ErrAppNotFound         = errors.New("no live connected app has that id")
	ErrNotLive             = errors.New("the token does not identify a live connected app")
	ErrRefreshReuse        = errors.New("a replaced refresh token was reused")
	ErrMissingPermission   = errors.New("the connected app was not granted that permission")
)

// Capabilities is what a connected app says it can handle, and it grants nothing
type Capabilities struct {
	AppVersion      string
	ProtocolVersion int
	Declared        []string
	AcceptedFormats []string
}

type StartInput struct {
	AppName string
	Name    string
	Capabilities
	Permissions []Permission
}

type AuthorizationInput struct {
	StartInput
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type Request struct {
	DeviceCode string
	UserCode   string
	VerifyURL  string
	ExpiresAt  time.Time
	Interval   time.Duration
}

type Authorization struct {
	URL       string
	UserCode  string
	ExpiresAt time.Time
}

type Pending struct {
	StartInput
	ExpiresAt     time.Time
	ApprovalToken string
}

type Redirect struct {
	URL string
}

type ConnectedApp struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	AppName string
	Name    string
	Capabilities
	Prefix      string
	Permissions []Permission
	ConnectedAt time.Time
	LastSeenAt  *time.Time
	RevokedAt   *time.Time
}

func (a ConnectedApp) Grants(permission Permission) bool {
	return slices.Contains(a.Permissions, permission)
}

func (a ConnectedApp) Declares(capability string) bool {
	return slices.Contains(a.Declared, capability)
}

type Credentials struct {
	ConnectedApp         ConnectedApp
	AccessToken          string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
}

const (
	maxNameLength       = 64
	maxVersionLength    = 64
	maxIdentifierLength = 64
	maxDeclaredItems    = 32
	maxRedirectLength   = 512
	protocolVersion     = 1

	codeAlphabet           = "BCDFGHJKLMNPQRSTVWXZ23456789"
	codeLength             = 8
	codeGroupSize          = 4
	secretBytes            = 32
	opaqueCodeLength       = 43
	maxUserCodeInputLength = 16

	accessTokenType  = string(credential.AppAccess)
	refreshTokenType = string(credential.AppRefresh)
)

var (
	capabilityPattern = regexp.MustCompile(`^[a-z][a-z0-9.-]*:[a-z][a-z0-9._-]*$`)
	formatPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	pkcePattern       = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

func validateStart(in StartInput) (StartInput, error) {
	app, err := validateText(in.AppName, maxNameLength)
	if err != nil {
		return StartInput{}, ErrInvalidName
	}
	name, err := validateText(in.Name, maxNameLength)
	if err != nil {
		return StartInput{}, ErrInvalidName
	}
	capabilities, err := validateCapabilities(in.Capabilities)
	if err != nil {
		return StartInput{}, err
	}
	permissions, err := canonicalPermissions(in.Permissions)
	if err != nil {
		return StartInput{}, err
	}
	return StartInput{
		AppName: app, Name: name, Capabilities: capabilities, Permissions: permissions,
	}, nil
}

func validateCapabilities(in Capabilities) (Capabilities, error) {
	version, err := checkAppVersion(in.AppVersion)
	if err != nil {
		return Capabilities{}, ErrInvalidCapabilities
	}
	if in.ProtocolVersion != protocolVersion || in.Declared == nil || in.AcceptedFormats == nil {
		return Capabilities{}, ErrInvalidCapabilities
	}
	declared, err := canonicalIdentifiers(in.Declared, capabilityPattern)
	if err != nil {
		return Capabilities{}, ErrInvalidCapabilities
	}
	formats, err := canonicalIdentifiers(in.AcceptedFormats, formatPattern)
	if err != nil {
		return Capabilities{}, ErrInvalidCapabilities
	}
	return Capabilities{
		AppVersion: version, ProtocolVersion: protocolVersion,
		Declared: declared, AcceptedFormats: formats,
	}, nil
}

// checkAppVersion trims a reported version, which may be empty, and refuses one that is not printable text of at most 64 characters
func checkAppVersion(raw string) (string, error) {
	version := strings.TrimSpace(raw)
	if version == "" {
		return "", nil
	}
	return validateText(version, maxVersionLength)
}

func validateText(raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || len([]rune(value)) > limit {
		return "", ErrInvalidCapabilities
	}
	for _, char := range value {
		if !unicode.IsPrint(char) {
			return "", ErrInvalidCapabilities
		}
	}
	return value, nil
}

func canonicalIdentifiers(values []string, pattern *regexp.Regexp) ([]string, error) {
	if len(values) > maxDeclaredItems {
		return nil, ErrInvalidCapabilities
	}
	result := make([]string, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if len(value) > maxIdentifierLength || !pattern.MatchString(value) {
			return nil, ErrInvalidCapabilities
		}
		if _, exists := seen[value]; exists {
			return nil, ErrInvalidCapabilities
		}
		seen[value] = struct{}{}
		result[index] = value
	}
	return result, nil
}

func canonicalPermissions(requested []Permission) ([]Permission, error) {
	granted := make([]Permission, 0, len(permissionOrder))
	for _, known := range permissionOrder {
		count := 0
		for _, asked := range requested {
			if asked == known {
				count++
			}
		}
		if count > 1 {
			return nil, ErrInvalidPermissions
		}
		if count == 1 {
			granted = append(granted, known)
		}
	}
	if len(granted) != len(requested) {
		return nil, ErrInvalidPermissions
	}
	return granted, nil
}

func permissionStrings(permissions []Permission) []string {
	stored := make([]string, len(permissions))
	for index, permission := range permissions {
		stored[index] = string(permission)
	}
	return stored
}

func permissionsFrom(stored []string) []Permission {
	permissions := make([]Permission, len(stored))
	for index, value := range stored {
		permissions[index] = Permission(value)
	}
	return permissions
}

func newCode(length int) (string, error) {
	limit := 256 / len(codeAlphabet) * len(codeAlphabet)
	code := make([]byte, 0, length)
	buffer := make([]byte, length)
	for len(code) < length {
		if _, err := rand.Read(buffer); err != nil {
			return "", fmt.Errorf("make code: %w", err)
		}
		for _, value := range buffer {
			if int(value) < limit && len(code) < length {
				code = append(code, codeAlphabet[int(value)%len(codeAlphabet)])
			}
		}
	}
	return string(code), nil
}

func FormatUserCode(code string) string {
	return code[:codeGroupSize] + "-" + code[codeGroupSize:]
}

func normalizeUserCode(raw string) (string, bool) {
	if len(raw) > maxUserCodeInputLength {
		return "", false
	}
	var code strings.Builder
	for _, char := range strings.ToUpper(raw) {
		if strings.ContainsRune(codeAlphabet, char) {
			code.WriteRune(char)
		} else if char != '-' && char != ' ' {
			return "", false
		}
	}
	if code.Len() != codeLength {
		return "", false
	}
	return code.String(), true
}

func newOpaqueCode() (string, []byte, error) {
	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", nil, fmt.Errorf("make secret: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(secret)
	return code, hashOf(code), nil
}

func opaqueCodeHash(code string) ([]byte, bool) {
	if len(code) != opaqueCodeLength {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil || len(raw) != secretBytes {
		return nil, false
	}
	return hashOf(code), true
}

func newCredential(secretType string) (token, prefix string, hash []byte, err error) {
	minted, err := credential.Mint(credential.Type(secretType))
	if err != nil {
		return "", "", nil, err
	}
	return minted.Value, minted.Prefix, minted.Hash, nil
}

func credentialHash(token, secretType string) ([]byte, bool) {
	read, ok := credential.Read(token, credential.Type(secretType))
	if !ok {
		return nil, false
	}
	return read.Hash, true
}

func validateAuthorization(in AuthorizationInput) (AuthorizationInput, error) {
	start, err := validateStart(in.StartInput)
	if err != nil {
		return AuthorizationInput{}, err
	}
	if !validLoopbackRedirect(in.RedirectURI) {
		return AuthorizationInput{}, ErrInvalidRedirect
	}
	if len(in.State) < 32 || len(in.State) > 128 || !pkcePattern.MatchString(in.State) {
		return AuthorizationInput{}, ErrInvalidPKCE
	}
	if in.CodeChallengeMethod != "S256" || !validChallenge(in.CodeChallenge) {
		return AuthorizationInput{}, ErrInvalidPKCE
	}
	return AuthorizationInput{
		StartInput: start, RedirectURI: in.RedirectURI, State: in.State,
		CodeChallenge: in.CodeChallenge, CodeChallengeMethod: "S256",
	}, nil
}

func validLoopbackRedirect(raw string) bool {
	if len(raw) > maxRedirectLength {
		return false
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Opaque != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path == "" {
		return false
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "::1" {
		return false
	}
	port, err := strconv.Atoi(parsed.Port())
	return err == nil && port > 0 && port <= 65535
}

func validChallenge(value string) bool {
	if len(value) != 43 || !pkcePattern.MatchString(value) {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(raw) == sha256.Size
}

func challengeMatches(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 || !pkcePattern.MatchString(verifier) {
		return false
	}
	digest := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(want), []byte(challenge)) == 1
}

func redirectWith(raw, key, value, state string) (string, error) {
	if !validLoopbackRedirect(raw) {
		return "", ErrInvalidRedirect
	}
	parsed, _ := url.Parse(raw)
	query := parsed.Query()
	query.Set(key, value)
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func hashOf(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
