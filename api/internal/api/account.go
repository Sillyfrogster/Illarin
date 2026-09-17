package api

import "github.com/google/uuid"

// Account is a person's account as every feature sees it
type Account struct {
	ID            uuid.UUID
	Handle        string
	Email         *string
	EmailVerified bool
	DiscordLinked bool
	HasPassword   bool
	Role          Role
}

type Role string

const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)
