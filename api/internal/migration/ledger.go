package migration

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Exception struct {
	Kind    string
	Subject string
	Detail  string
	AssetID *uuid.UUID
}

type Ledger struct {
	policy  map[string]format.AnomalyDisposition
	entries []Exception
}

func NewLedger(anomalies []format.AnomalyDeclaration) (*Ledger, error) {
	if err := format.ValidateAnomalies(anomalies); err != nil {
		return nil, fmt.Errorf("anomaly policy: %w", err)
	}
	policy := make(map[string]format.AnomalyDisposition, len(anomalies))
	for _, anomaly := range anomalies {
		policy[anomaly.Kind] = anomaly.Disposition
	}
	return &Ledger{policy: policy}, nil
}

func (l *Ledger) Raise(entry Exception) error {
	if entry.Kind == "" || entry.Subject == "" || entry.Detail == "" {
		return fmt.Errorf("anomaly %q needs a kind, a subject and a detail", entry.Kind)
	}
	disposition, declared := l.policy[entry.Kind]
	if !declared {
		return fmt.Errorf(
			"anomaly %q on %s was not classified before the run: %s",
			entry.Kind, entry.Subject, entry.Detail,
		)
	}
	if disposition == format.AnomalyFatal {
		return fmt.Errorf("%s on %s: %s", entry.Kind, entry.Subject, entry.Detail)
	}
	l.entries = append(l.entries, entry)
	return nil
}

func (l *Ledger) Entries() []Exception {
	return l.entries
}

func (l *Ledger) Count(kind string) int {
	total := 0
	for _, entry := range l.entries {
		if entry.Kind == kind {
			total++
		}
	}
	return total
}

func (l *Ledger) Persist(ctx context.Context, tx db.DBTX) error {
	queries := db.New(tx)
	for _, entry := range l.entries {
		asset := pgtype.UUID{}
		if entry.AssetID != nil {
			asset = pgtype.UUID{Bytes: *entry.AssetID, Valid: true}
		}
		if err := queries.InsertMigrationException(ctx, db.InsertMigrationExceptionParams{
			ID:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Kind:    entry.Kind,
			Subject: entry.Subject,
			Detail:  entry.Detail,
			AssetID: asset,
		}); err != nil {
			return fmt.Errorf("write ledger entry %s on %s: %w", entry.Kind, entry.Subject, err)
		}
	}
	return nil
}
