package http

import (
	"time"

	"github.com/google/uuid"
)

type AcceptedTargets = []ExportTargetId

type AccessToken = string

type ApplicationName = string

type ApplicationVersion = string

type ApprovalToken = string

type AuthorizationCode = string

type CapabilityId = string

type DeviceCode = string

type DeviceLinkDecision struct {
	ApprovalToken ApprovalToken `json:"approvalToken"`
}

type ExchangeLinkAuthorization struct {
	AuthorizationCode AuthorizationCode `json:"authorizationCode"`
	CodeVerifier      string            `json:"codeVerifier"`
	RedirectUri       string            `json:"redirectUri"`
}

type ExportTargetId = string

type InstanceCapabilities = []CapabilityId

type InstanceName = string

type InstanceTokenGrant struct {
	AccessToken          AccessToken    `json:"accessToken"`
	AccessTokenExpiresAt time.Time      `json:"accessTokenExpiresAt"`
	Instance             LinkedInstance `json:"instance"`
	RefreshToken         RefreshToken   `json:"refreshToken"`
}

type LinkAuthorization struct {
	AuthorizationUrl string    `json:"authorizationUrl"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type LinkProtocolVersion int

const (
	LinkProtocolVersionN1 LinkProtocolVersion = 1
)

type LinkRedirect struct {
	RedirectUrl string `json:"redirectUrl"`
}

type LinkRequest struct {
	DeviceCode      DeviceCode `json:"deviceCode"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	Interval        int        `json:"interval"`
	UserCode        UserCode   `json:"userCode"`
	VerificationUrl string     `json:"verificationUrl"`
}

type LinkedInstance struct {
	AcceptedTargets    []string   `json:"acceptedTargets"`
	ApplicationName    string     `json:"applicationName"`
	ApplicationVersion *string    `json:"applicationVersion,omitempty"`
	Capabilities       []string   `json:"capabilities"`
	Id                 uuid.UUID  `json:"id"`
	InstanceName       string     `json:"instanceName"`
	LastSeenAt         *time.Time `json:"lastSeenAt"`
	LinkedAt           time.Time  `json:"linkedAt"`
	Prefix             string     `json:"prefix"`
	ProtocolVersion    *int       `json:"protocolVersion"`
	RevokedAt          *time.Time `json:"revokedAt"`
	Scopes             []Scope    `json:"scopes"`
}

type LinkedInstanceList struct {
	Items []ManagedInstance `json:"items"`
}

type LinkedLinkPollResult struct {
	AccessToken          AccessToken                `json:"accessToken"`
	AccessTokenExpiresAt time.Time                  `json:"accessTokenExpiresAt"`
	Instance             LinkedInstance             `json:"instance"`
	RefreshToken         RefreshToken               `json:"refreshToken"`
	Status               LinkedLinkPollResultStatus `json:"status"`
}

type LinkedLinkPollResultStatus string

const (
	Linked LinkedLinkPollResultStatus = "linked"
)

type ManagedInstance struct {
	AcceptedTargets    []string   `json:"acceptedTargets"`
	ApplicationName    string     `json:"applicationName"`
	ApplicationVersion *string    `json:"applicationVersion,omitempty"`
	Capabilities       []string   `json:"capabilities"`
	Id                 uuid.UUID  `json:"id"`
	Installed          int        `json:"installed"`
	InstanceName       string     `json:"instanceName"`
	LastSeenAt         *time.Time `json:"lastSeenAt"`
	LinkedAt           time.Time  `json:"linkedAt"`
	Prefix             string     `json:"prefix"`
	ProtocolVersion    *int       `json:"protocolVersion"`
	RevokedAt          *time.Time `json:"revokedAt"`
	Scopes             []Scope    `json:"scopes"`
	UpdatesAvailable   int        `json:"updatesAvailable"`
}

type PendingDeviceLink struct {
	AcceptedTargets    AcceptedTargets      `json:"acceptedTargets"`
	ApplicationName    ApplicationName      `json:"applicationName"`
	ApplicationVersion *ApplicationVersion  `json:"applicationVersion,omitempty"`
	ApprovalToken      ApprovalToken        `json:"approvalToken"`
	Capabilities       InstanceCapabilities `json:"capabilities"`
	ExpiresAt          time.Time            `json:"expiresAt"`
	InstanceName       InstanceName         `json:"instanceName"`
	ProtocolVersion    LinkProtocolVersion  `json:"protocolVersion"`
	Scopes             Scopes               `json:"scopes"`
}

type PendingLink struct {
	AcceptedTargets    AcceptedTargets      `json:"acceptedTargets"`
	ApplicationName    ApplicationName      `json:"applicationName"`
	ApplicationVersion *ApplicationVersion  `json:"applicationVersion,omitempty"`
	Capabilities       InstanceCapabilities `json:"capabilities"`
	ExpiresAt          time.Time            `json:"expiresAt"`
	InstanceName       InstanceName         `json:"instanceName"`
	ProtocolVersion    LinkProtocolVersion  `json:"protocolVersion"`
	Scopes             Scopes               `json:"scopes"`
}

type PendingLinkPollResult struct {
	Status PendingLinkPollResultStatus `json:"status"`
}

type PendingLinkPollResultStatus string

const (
	PendingLinkPollResultStatusPending PendingLinkPollResultStatus = "pending"
)

type PollLinkRequest struct {
	DeviceCode DeviceCode `json:"deviceCode"`
}

type RefreshInstanceToken struct {
	RefreshToken RefreshToken `json:"refreshToken"`
}

type RefreshToken = string

type RequestCode = string

type Scope string

const (
	AssetReceive Scope = "asset:receive"
	LibrarySync  Scope = "library:sync"
)

type Scopes = []Scope

type StartLinkAuthorization struct {
	AcceptedTargets     AcceptedTargets                           `json:"acceptedTargets"`
	ApplicationName     ApplicationName                           `json:"applicationName"`
	ApplicationVersion  *ApplicationVersion                       `json:"applicationVersion,omitempty"`
	Capabilities        InstanceCapabilities                      `json:"capabilities"`
	CodeChallenge       string                                    `json:"codeChallenge"`
	CodeChallengeMethod StartLinkAuthorizationCodeChallengeMethod `json:"codeChallengeMethod"`
	InstanceName        InstanceName                              `json:"instanceName"`
	ProtocolVersion     LinkProtocolVersion                       `json:"protocolVersion"`
	RedirectUri         string                                    `json:"redirectUri"`
	Scopes              Scopes                                    `json:"scopes"`
	State               string                                    `json:"state"`
}

type StartLinkAuthorizationCodeChallengeMethod string

const (
	S256 StartLinkAuthorizationCodeChallengeMethod = "S256"
)

type StartLinkRequest struct {
	AcceptedTargets    AcceptedTargets      `json:"acceptedTargets"`
	ApplicationName    ApplicationName      `json:"applicationName"`
	ApplicationVersion *ApplicationVersion  `json:"applicationVersion,omitempty"`
	Capabilities       InstanceCapabilities `json:"capabilities"`
	InstanceName       InstanceName         `json:"instanceName"`
	ProtocolVersion    LinkProtocolVersion  `json:"protocolVersion"`
	Scopes             Scopes               `json:"scopes"`
}

type UpdateInstance struct {
	AcceptedTargets    AcceptedTargets      `json:"acceptedTargets"`
	ApplicationVersion *ApplicationVersion  `json:"applicationVersion,omitempty"`
	Capabilities       InstanceCapabilities `json:"capabilities"`
	ProtocolVersion    LinkProtocolVersion  `json:"protocolVersion"`
}

type UserCode = string
