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
	Destinations *[]uuid.UUID
	Ping         []uuid.UUID
	Note         string
}

var ErrRoleRefused = errors.New("the destination has no role this post may ping")

type DestinationPolicy struct {
	Allowed  *[]uuid.UUID
	Defaults []uuid.UUID
}

func (s *Service) AppChoices(ctx context.Context, appID uuid.UUID) ([]Choice, error) {
	return choicesFrom(ctx, s.pool, `
		select destination.id, destination.name, destination.kind, destination.state,
		       destination.events, destination.role_name, allowed.by_default
		  from publication_app_destinations allowed
		  join publication_destinations destination on destination.id = allowed.destination_id
		 where allowed.app_id = $1
		 order by destination.name, destination.created_at
	`, appID)
}

func (s *Service) GrantChoices(ctx context.Context, grantID uuid.UUID) ([]Choice, error) {
	var overridden bool
	var appID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		select destinations_overridden, app_id from publication_grants where id = $1
	`, grantID).Scan(&overridden, &appID)
	if err != nil {
		return nil, fmt.Errorf("read the destination policy behind a grant: %w", err)
	}
	if !overridden {
		return s.AppChoices(ctx, appID)
	}
	return choicesFrom(ctx, s.pool, `
		select destination.id, destination.name, destination.kind, destination.state,
		       destination.events, destination.role_name, allowed.by_default
		  from publication_grant_destinations allowed
		  join publication_destinations destination on destination.id = allowed.destination_id
		 where allowed.grant_id = $1
		 order by destination.name, destination.created_at
	`, grantID)
}

func (s *Service) PostChoices(ctx context.Context, grantID *uuid.UUID) ([]Choice, error) {
	if grantID != nil {
		return s.GrantChoices(ctx, *grantID)
	}
	return choicesFrom(ctx, s.pool, `
		select destination.id, destination.name, destination.kind, destination.state,
		       destination.events, destination.role_name, false
		  from publication_destinations destination
		 order by destination.name, destination.created_at
	`)
}

func (s *Service) checkPolicy(
	ctx context.Context,
	in DestinationPolicy,
) (map[uuid.UUID]bool, error) {
	allowed := make(map[uuid.UUID]bool)
	if in.Allowed == nil {
		if len(in.Defaults) > 0 {
			return nil, FieldError{
				Field:   "defaultDestinationIds",
				Message: "A default belongs to a set this policy names.",
			}
		}
		return allowed, nil
	}
	for _, id := range *in.Allowed {
		if _, err := s.Destination(ctx, id); err != nil {
			return nil, err
		}
		allowed[id] = false
	}
	for _, id := range in.Defaults {
		if _, held := allowed[id]; !held {
			return nil, FieldError{
				Field:   "defaultDestinationIds",
				Message: "A default has to be one of the allowed destinations.",
			}
		}
		allowed[id] = true
	}
	return allowed, nil
}

type Sending struct {
	Choice
	Ping bool
}

func (s *Service) Chosen(
	ctx context.Context,
	grantID *uuid.UUID,
	in Announcement,
) ([]Sending, string, error) {
	note := oneParagraph(in.Note)
	if len(note) > noteLimit {
		return nil, "", FieldError{
			Field:   "note",
			Message: fmt.Sprintf("Keep the announcement note under %d characters.", noteLimit),
		}
	}
	allowed, err := s.PostChoices(ctx, grantID)
	if err != nil {
		return nil, "", err
	}
	picked := defaultsAmong(allowed)
	if in.Destinations != nil {
		if picked, err = named(allowed, *in.Destinations); err != nil {
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
			return nil, ErrDestinationRefused
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
		if one.State == DestinationActive {
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
		return nil, fmt.Errorf("read the destinations a post may send to: %w", err)
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
			&one.ID, &one.Name, &one.Kind, &one.State, &one.Events, &role, &one.Ping,
		)
		if err != nil {
			return nil, fmt.Errorf("read a destination a post captured: %w", err)
		}
		if role != nil {
			one.Role = *role
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the destinations a post captured: %w", err)
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
			&one.ID, &one.Name, &one.Kind, &one.State, &one.Events, &role, &one.ByDefault,
		)
		if err != nil {
			return nil, fmt.Errorf("read a destination a post may send to: %w", err)
		}
		if role != nil {
			one.Role = *role
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the destinations a post may send to: %w", err)
	}
	return found, nil
}

func (s *Service) SetGrantDestinations(ctx context.Context, actor, grantID, appID, holderID uuid.UUID, in DestinationPolicy) error {
	allowed, err := s.checkPolicy(ctx, in)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin grant destination policy: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		delete from publication_grant_destinations where grant_id = $1
	`, grantID); err != nil {
		return fmt.Errorf("clear the grant destination policy: %w", err)
	}
	_, err = tx.Exec(ctx, `
		update publication_grants set destinations_overridden = $2 where id = $1
	`, grantID, in.Allowed != nil)
	if err != nil {
		return fmt.Errorf("record the grant destination override: %w", err)
	}
	for id, byDefault := range allowed {
		_, err := tx.Exec(ctx, `
			insert into publication_grant_destinations (grant_id, destination_id, by_default)
			values ($1, $2, $3)
		`, grantID, id, byDefault)
		if err != nil {
			return fmt.Errorf("allow a grant destination: %w", err)
		}
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "grant.destinations.set",
		AppID: &appID, GrantID: &grantID, SubjectID: &holderID,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit grant destination policy: %w", err)
	}
	return nil
}
