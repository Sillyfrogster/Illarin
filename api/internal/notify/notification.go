package notify

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Type names one type of notification in the closed set Illarin sends.
type Type string

const (
	WorkWithheld      Type = "work_withheld"
	WorkRestored      Type = "work_restored"
	WorkUpdated       Type = "work_updated"
	ProfileRestricted Type = "profile_restricted"
	ProfileRestored   Type = "profile_restored"
)

// Words is what a notification shows, kept as it read when the change happened.
type Words struct {
	WorkName     string `json:"workName,omitempty"`
	Reason       string `json:"reason,omitempty"`
	UpdateNumber int    `json:"updateNumber,omitempty"`
	VersionLabel string `json:"versionLabel,omitempty"`
	Summary      string `json:"summary,omitempty"`
}

// Event is one change to tell people about. Without an account, the fan-out works out who hears from the work.
type Event struct {
	Type    Type
	Account *uuid.UUID
	Work    *uuid.UUID
	Words   Words
}

// Record writes the event in the caller's transaction, so it exists exactly when the change commits.
func Record(ctx context.Context, tx pgx.Tx, event Event) error {
	words, err := json.Marshal(event.Words)
	if err != nil {
		return fmt.Errorf("encode the notification's words: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into notification_events (id, type, account_id, work_id, words)
		values ($1, $2, $3, $4, $5)
	`, uuid.New(), event.Type, event.Account, event.Work, words); err != nil {
		return fmt.Errorf("record a notification event: %w", err)
	}
	return nil
}
