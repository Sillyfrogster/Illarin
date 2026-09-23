package account

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

type NSFWPreference string

const (
	NSFWHidden  NSFWPreference = "hidden"
	NSFWBlurred NSFWPreference = "blurred"
	NSFWShown   NSFWPreference = "shown"
)

func (p NSFWPreference) valid() bool {
	return p == NSFWHidden || p == NSFWBlurred || p == NSFWShown
}

// AppAny is the app preference of a reader who wants every app's work
const AppAny = "any"

// ValidApp says whether an app preference names an app in the registry or any
func ValidApp(app string) bool {
	return app == AppAny || format.KnownApp(app)
}

// Preferences holds what a reader chose about browse, where a nil App means they have not said
type Preferences struct {
	App  *string
	NSFW NSFWPreference
}

// SignUpInput leaves App and NSFW empty when the reader skipped them
type SignUpInput struct {
	Email    string
	Password string
	Handle   string
	App      string
	NSFW     NSFWPreference
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
