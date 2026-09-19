package connect

import (
	"time"

	"github.com/google/uuid"
)

type Permission string

const (
	PermissionReceiveWorks Permission = "work:receive"
	PermissionSyncLibrary  Permission = "library:sync"
)

type StartConnectionRequest struct {
	AppName         string       `json:"appName"`
	Name            string       `json:"name"`
	AppVersion      *string      `json:"appVersion,omitempty"`
	ProtocolVersion int          `json:"protocolVersion" tstype:"1,required"`
	Capabilities    []string     `json:"capabilities"`
	AcceptedFormats []string     `json:"acceptedFormats"`
	Permissions     []Permission `json:"permissions"`
}

type ConnectionRequest struct {
	DeviceCode      string    `json:"deviceCode"`
	UserCode        string    `json:"userCode"`
	VerificationUrl string    `json:"verificationUrl"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Interval        int       `json:"interval"`
}

type StartConnectionAuthorization struct {
	AppName             string       `json:"appName"`
	Name                string       `json:"name"`
	AppVersion          *string      `json:"appVersion,omitempty"`
	ProtocolVersion     int          `json:"protocolVersion" tstype:"1,required"`
	Capabilities        []string     `json:"capabilities"`
	AcceptedFormats     []string     `json:"acceptedFormats"`
	Permissions         []Permission `json:"permissions"`
	RedirectUri         string       `json:"redirectUri"`
	State               string       `json:"state"`
	CodeChallenge       string       `json:"codeChallenge"`
	CodeChallengeMethod string       `json:"codeChallengeMethod" tstype:"'S256',required"`
}

type ConnectionAuthorization struct {
	AuthorizationUrl string    `json:"authorizationUrl"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type PollConnectionRequest struct {
	DeviceCode string `json:"deviceCode"`
}

type PendingPoll struct {
	Status string `json:"status" tstype:"'pending',required"`
}

type ConnectedPoll struct {
	Status               string             `json:"status" tstype:"'connected',required"`
	AccessToken          string             `json:"accessToken"`
	AccessTokenExpiresAt time.Time          `json:"accessTokenExpiresAt"`
	RefreshToken         string             `json:"refreshToken"`
	ConnectedApp         ConnectedAppDetail `json:"connectedApp"`
}

type PendingConnection struct {
	AppName         string       `json:"appName"`
	Name            string       `json:"name"`
	AppVersion      *string      `json:"appVersion,omitempty"`
	ProtocolVersion int          `json:"protocolVersion" tstype:"1,required"`
	Capabilities    []string     `json:"capabilities"`
	AcceptedFormats []string     `json:"acceptedFormats"`
	Permissions     []Permission `json:"permissions"`
	ExpiresAt       time.Time    `json:"expiresAt"`
}

type PendingCodeConnection struct {
	AppName         string       `json:"appName"`
	Name            string       `json:"name"`
	AppVersion      *string      `json:"appVersion,omitempty"`
	ProtocolVersion int          `json:"protocolVersion" tstype:"1,required"`
	Capabilities    []string     `json:"capabilities"`
	AcceptedFormats []string     `json:"acceptedFormats"`
	Permissions     []Permission `json:"permissions"`
	ExpiresAt       time.Time    `json:"expiresAt"`
	ApprovalToken   string       `json:"approvalToken"`
}

type ConnectionDecision struct {
	ApprovalToken string `json:"approvalToken"`
}

type ConnectionRedirect struct {
	RedirectUrl string `json:"redirectUrl"`
}

type ExchangeConnectionAuthorization struct {
	AuthorizationCode string `json:"authorizationCode"`
	CodeVerifier      string `json:"codeVerifier"`
	RedirectUri       string `json:"redirectUri"`
}

type RefreshAppCredentials struct {
	RefreshToken string `json:"refreshToken"`
}

type AppCredentials struct {
	AccessToken          string             `json:"accessToken"`
	AccessTokenExpiresAt time.Time          `json:"accessTokenExpiresAt"`
	RefreshToken         string             `json:"refreshToken"`
	ConnectedApp         ConnectedAppDetail `json:"connectedApp"`
}

type ConnectedAppDetail struct {
	Id              uuid.UUID    `json:"id"`
	AppName         string       `json:"appName"`
	Name            string       `json:"name"`
	AppVersion      *string      `json:"appVersion,omitempty"`
	ProtocolVersion *int         `json:"protocolVersion" tstype:"number | null,required"`
	Capabilities    []string     `json:"capabilities"`
	AcceptedFormats []string     `json:"acceptedFormats"`
	Prefix          string       `json:"prefix"`
	Permissions     []Permission `json:"permissions"`
	ConnectedAt     time.Time    `json:"connectedAt"`
	LastSeenAt      *time.Time   `json:"lastSeenAt" tstype:"string | null,required"`
	RevokedAt       *time.Time   `json:"revokedAt" tstype:"string | null,required"`
}

type ManagedConnectedApp struct {
	Id               uuid.UUID    `json:"id"`
	AppName          string       `json:"appName"`
	Name             string       `json:"name"`
	AppVersion       *string      `json:"appVersion,omitempty"`
	ProtocolVersion  *int         `json:"protocolVersion" tstype:"number | null,required"`
	Capabilities     []string     `json:"capabilities"`
	AcceptedFormats  []string     `json:"acceptedFormats"`
	Prefix           string       `json:"prefix"`
	Permissions      []Permission `json:"permissions"`
	ConnectedAt      time.Time    `json:"connectedAt"`
	LastSeenAt       *time.Time   `json:"lastSeenAt" tstype:"string | null,required"`
	RevokedAt        *time.Time   `json:"revokedAt" tstype:"string | null,required"`
	Installed        int          `json:"installed"`
	UpdatesAvailable int          `json:"updatesAvailable"`
}

type ConnectedAppList struct {
	Items []ManagedConnectedApp `json:"items"`
}

type UpdateCapabilities struct {
	AppVersion      *string  `json:"appVersion,omitempty"`
	ProtocolVersion int      `json:"protocolVersion" tstype:"1,required"`
	Capabilities    []string `json:"capabilities"`
	AcceptedFormats []string `json:"acceptedFormats"`
}
