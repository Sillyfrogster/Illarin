package blog

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const noteLimit = 500

type Announcement struct {
	Integrations *[]uuid.UUID
	Ping         []uuid.UUID
	Note         string
}

var ErrRoleRefused = errors.New("the integration has no role this post may ping")

// PostChoices lists every integration a post may announce to, each chosen unless the writer unticks it
func (s *Service) PostChoices(ctx context.Context) ([]Choice, error) {
	return choicesFrom(ctx, s.pool, `
		select integration.id, integration.name, integration.type, integration.state,
		       integration.announcements, integration.role_name, true
		  from blog_integrations integration
		 order by integration.name, integration.created_at
	`)
}

type Sending struct {
	Choice
	Ping bool
}

func (s *Service) Chosen(
	ctx context.Context,
	in Announcement,
) ([]Sending, string, error) {
	note := oneParagraph(in.Note)
	if len(note) > noteLimit {
		return nil, "", FieldError{
			Field:   "note",
			Message: fmt.Sprintf("Keep the announcement note under %d characters.", noteLimit),
		}
	}
	allowed, err := s.PostChoices(ctx)
	if err != nil {
		return nil, "", err
	}
	picked := defaultsAmong(allowed)
	if in.Integrations != nil {
		if picked, err = named(allowed, *in.Integrations); err != nil {
			return nil, "", err
		}
	}
	going, err := pinged(ActiveAmong(picked), in.Ping)
	if err != nil {
		return nil, "", err
	}
	return going, note, nil
}

func named(allowed []Choice, wanted []uuid.UUID) ([]Choice, error) {
	byID := make(map[uuid.UUID]Choice, len(allowed))
	for _, one := range allowed {
		byID[one.ID] = one
	}
	picked := make([]Choice, 0, len(wanted))
	seen := make(map[uuid.UUID]bool, len(wanted))
	for _, id := range wanted {
		one, held := byID[id]
		if !held {
			return nil, ErrIntegrationRefused
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		picked = append(picked, one)
	}
	return picked, nil
}

func pinged(picked []Choice, wanted []uuid.UUID) ([]Sending, error) {
	asked := make(map[uuid.UUID]bool, len(wanted))
	for _, id := range wanted {
		asked[id] = true
	}
	going := make([]Sending, 0, len(picked))
	for _, one := range picked {
		ping := asked[one.ID]
		if ping && one.Role == "" {
			return nil, ErrRoleRefused
		}
		delete(asked, one.ID)
		going = append(going, Sending{Choice: one, Ping: ping})
	}
	if len(asked) > 0 {
		return nil, ErrRoleRefused
	}
	return going, nil
}

func ActiveAmong(held []Choice) []Choice {
	ready := make([]Choice, 0, len(held))
	for _, one := range held {
		if one.State == IntegrationActive {
			ready = append(ready, one)
		}
	}
	return ready
}

func defaultsAmong(held []Choice) []Choice {
	chosen := make([]Choice, 0, len(held))
	for _, one := range held {
		if one.ByDefault {
			chosen = append(chosen, one)
		}
	}
	return chosen
}

func choicesFrom(
	ctx context.Context,
	pool *pgxpool.Pool,
	query string,
	args ...any,
) ([]Choice, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read the integrations a post may send to: %w", err)
	}
	return collectChoices(rows)
}

func CollectSending(rows pgx.Rows) ([]Sending, error) {
	defer rows.Close()
	found := make([]Sending, 0, 4)
	for rows.Next() {
		var one Sending
		var role *string
		err := rows.Scan(
			&one.ID, &one.Name, &one.Type, &one.State, &one.Announcements, &role, &one.Ping,
		)
		if err != nil {
			return nil, fmt.Errorf("read a integration a post captured: %w", err)
		}
		if role != nil {
			one.Role = *role
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the integrations a post captured: %w", err)
	}
	return found, nil
}

func collectChoices(rows pgx.Rows) ([]Choice, error) {
	defer rows.Close()
	found := make([]Choice, 0, 4)
	for rows.Next() {
		var one Choice
		var role *string
		err := rows.Scan(
			&one.ID, &one.Name, &one.Type, &one.State, &one.Announcements, &role, &one.ByDefault,
		)
		if err != nil {
			return nil, fmt.Errorf("read a integration a post may send to: %w", err)
		}
		if role != nil {
			one.Role = *role
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the integrations a post may send to: %w", err)
	}
	return found, nil
}
