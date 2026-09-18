package account

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/google/uuid"
)

type NSFWPreference string

const (
	NSFWHidden  NSFWPreference = "hidden"
	NSFWBlurred NSFWPreference = "blurred"
	NSFWShown   NSFWPreference = "shown"
)

type SignUpInput struct {
	Email    string
	Password string
	Handle   string
}

type DiscordProfile struct {
	Subject       string
	Username      string
	DisplayName   string
	AvatarURL     string
	BannerURL     string
	Email         string
	EmailVerified bool
}

type DiscordIntent string

const (
	DiscordSignIn DiscordIntent = "sign-in"
	DiscordAttach DiscordIntent = "attach"
)

type DiscordAuthorization struct {
	URL     string
	State   string
	Expires time.Time
}

type DiscordCompletion struct {
	Account        api.Account
	SessionToken   string
	SessionExpires time.Time
	Intent         DiscordIntent
}

type CreatorListing struct {
	ID                             uuid.UUID
	Handle                         string
	ShowNSFWContributionsOnProfile bool
}
