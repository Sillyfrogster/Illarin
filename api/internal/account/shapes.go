package account

import (
	"github.com/google/uuid"
)

func (e NsfwVisibilityRequestVisibility) Valid() bool {
	switch e {
	case NsfwVisibilityRequestVisibilityBlurred:
		return true
	case NsfwVisibilityRequestVisibilityHidden:
		return true
	case NsfwVisibilityRequestVisibilityShown:
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

type NsfwVisibilityRequest struct {
	Visibility NsfwVisibilityRequestVisibility `json:"visibility"`
}

type NsfwVisibilityRequestVisibility string

const (
	NsfwVisibilityRequestVisibilityBlurred NsfwVisibilityRequestVisibility = "blurred"
	NsfwVisibilityRequestVisibilityHidden  NsfwVisibilityRequestVisibility = "hidden"
	NsfwVisibilityRequestVisibilityShown   NsfwVisibilityRequestVisibility = "shown"
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
