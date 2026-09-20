package account

import (
	"github.com/google/uuid"
)

func (e NsfwPreferenceRequestPreference) Valid() bool {
	switch e {
	case NsfwPreferenceRequestPreferenceBlurred:
		return true
	case NsfwPreferenceRequestPreferenceHidden:
		return true
	case NsfwPreferenceRequestPreferenceShown:
		return true
	default:
		return false
	}
}

type Account struct {
	DiscordLinked bool        `json:"discordLinked"`
	Email         *string     `json:"email" tstype:"string | null,required"`
	EmailVerified bool        `json:"emailVerified"`
	Handle        string      `json:"handle"`
	HasPassword   bool        `json:"hasPassword"`
	Id            uuid.UUID   `json:"id"`
	Role          AccountRole `json:"role"`
}

type AccountRole string

const (
	AccountRoleAdmin     AccountRole = "admin"
	AccountRoleModerator AccountRole = "moderator"
	AccountRoleUser      AccountRole = "user"
)

type ChangeEmailRequest struct {
	Email string `json:"email"`
}

type CompletePasswordResetRequest struct {
	Password string `json:"password"`
	Token    string `json:"token"`
}

type NsfwPreferenceRequest struct {
	Preference NsfwPreferenceRequestPreference `json:"preference"`
}

type NsfwPreferenceRequestPreference string

const (
	NsfwPreferenceRequestPreferenceBlurred NsfwPreferenceRequestPreference = "blurred"
	NsfwPreferenceRequestPreferenceHidden  NsfwPreferenceRequestPreference = "hidden"
	NsfwPreferenceRequestPreferenceShown   NsfwPreferenceRequestPreference = "shown"
)

type PasswordRequest struct {
	Password string `json:"password"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type RenameHandleRequest struct {
	Handle string `json:"handle"`
}

type SessionState struct {
	PublicationAuthority bool     `json:"publicationAuthority"`
	User                 *Account `json:"user" tstype:"Account | null,required"`
	Writer               bool     `json:"writer"`
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignUpRequest struct {
	Email    string `json:"email"`
	Handle   string `json:"handle"`
	Password string `json:"password"`
}

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

type BeginDiscordParams struct {
	Intent   *BeginDiscordParamsIntent `json:"intent,omitempty"`
	ReturnTo *string                   `json:"returnTo,omitempty"`
}

type CompleteDiscordParams struct {
	State string  `json:"state"`
	Code  *string `json:"code,omitempty"`
	Error *string `json:"error,omitempty"`
}

type BeginDiscordParamsIntent string

const (
	BeginDiscordParamsIntentAttach BeginDiscordParamsIntent = "attach"
	BeginDiscordParamsIntentSignIn BeginDiscordParamsIntent = "sign-in"
)
