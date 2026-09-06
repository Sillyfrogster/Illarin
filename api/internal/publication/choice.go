package publication

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// noteLimit is the longest announcement note one transition may carry.
const noteLimit = 500

// Announcement is the delivery choice one public transition captures. A nil
// destination list accepts what the policy already defaults to; an empty one
// is a deliberately quiet publication.
type Announcement struct {
	Destinations *[]uuid.UUID
	Note         string
}

// DestinationPolicy is the allowed and default set an app or a grant carries.
// A nil Allowed on a grant means the grant follows its app.
type DestinationPolicy struct {
	Allowed  *[]uuid.UUID
	Defaults []uuid.UUID
}

// AppChoices answers the safe destination identities one app allows.
func (s *Service) AppChoices(ctx context.Context, appID uuid.UUID) ([]Choice, error) {
	return choicesFrom(ctx, s.pool, `
		select destination.id, destination.name, destination.kind, destination.state,
		       allowed.by_default
		  from publication_app_destinations allowed
		  join publication_destinations destination on destination.id = allowed.destination_id
		 where allowed.app_id = $1
		 order by destination.name, destination.created_at
	`, appID)
}

// GrantChoices answers the safe destination identities one grant may send to,
// which is the grant's own set where it has one and its app's set otherwise.
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
		       allowed.by_default
		  from publication_grant_destinations allowed
		  join publication_destinations destination on destination.id = allowed.destination_id
		 where allowed.grant_id = $1
		 order by destination.name, destination.created_at
	`, grantID)
}

// PostChoices answers what one post may send to. A post published under a grant
// is bound by that grant; a post an admin owns outright may reach any active
// destination, because an admin holds post authority without holding a grant.
func (s *Service) PostChoices(ctx context.Context, grantID *uuid.UUID) ([]Choice, error) {
	if grantID != nil {
		return s.GrantChoices(ctx, *grantID)
	}
	return choicesFrom(ctx, s.pool, `
		select destination.id, destination.name, destination.kind, destination.state, false
		  from publication_destinations destination
		 order by destination.name, destination.created_at
	`)
}

// PostDestinations answers the destinations one post may send to, to the
// contributor who owns it and to an admin.
func (s *Service) PostDestinations(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
) ([]Choice, error) {
	found, err := s.post(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, found); err != nil {
		return nil, err
	}
	allowed, err := s.PostChoices(ctx, found.GrantID)
	if err != nil {
		return nil, err
	}
	return activeAmong(allowed), nil
}

// SetAppDestinations records the baseline every grant on one app follows.
func (s *Service) SetAppDestinations(
	ctx context.Context,
	actor uuid.UUID,
	appID uuid.UUID,
	in DestinationPolicy,
) error {
	if _, err := s.app(ctx, appID); err != nil {
		return err
	}
	allowed, err := s.checkPolicy(ctx, in)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin app destination policy: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		delete from publication_app_destinations where app_id = $1
	`, appID); err != nil {
		return fmt.Errorf("clear the app destination policy: %w", err)
	}
	for id, byDefault := range allowed {
		_, err := tx.Exec(ctx, `
			insert into publication_app_destinations (app_id, destination_id, by_default)
			values ($1, $2, $3)
		`, appID, id, byDefault)
		if err != nil {
			return fmt.Errorf("allow an app destination: %w", err)
		}
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "app.destinations.set", AppID: &appID,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit app destination policy: %w", err)
	}
	return nil
}

// SetGrantDestinations narrows one contributor to its own set, or clears the
// override so the grant follows its app again.
func (s *Service) SetGrantDestinations(
	ctx context.Context,
	actor uuid.UUID,
	grantID uuid.UUID,
	in DestinationPolicy,
) error {
	current, err := s.grant(ctx, grantID)
	if err != nil {
		return err
	}
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
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "grant.destinations.set",
		AppID: &current.App.ID, GrantID: &grantID, SubjectID: &current.Holder.ID,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit grant destination policy: %w", err)
	}
	return nil
}

// checkPolicy refuses a set naming a destination that does not exist and a
// default outside the set it belongs to.
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

// chosen answers the destinations one transition will send to, having refused
// every identity the post is not allowed to reach.
func (s *Service) chosen(
	ctx context.Context,
	grantID *uuid.UUID,
	in Announcement,
) ([]Choice, string, error) {
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
	if in.Destinations == nil {
		return activeAmong(defaultsAmong(allowed)), note, nil
	}
	byID := make(map[uuid.UUID]Choice, len(allowed))
	for _, one := range allowed {
		byID[one.ID] = one
	}
	picked := make([]Choice, 0, len(*in.Destinations))
	seen := make(map[uuid.UUID]bool, len(*in.Destinations))
	for _, id := range *in.Destinations {
		one, held := byID[id]
		if !held {
			return nil, "", ErrDestinationRefused
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		picked = append(picked, one)
	}
	return activeAmong(picked), note, nil
}

// activeAmong drops a destination that is not ready to receive anything, so a
// disabled endpoint never becomes delivery work nobody asked for.
func activeAmong(held []Choice) []Choice {
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

func collectChoices(rows pgx.Rows) ([]Choice, error) {
	defer rows.Close()
	found := make([]Choice, 0, 4)
	for rows.Next() {
		var one Choice
		err := rows.Scan(&one.ID, &one.Name, &one.Kind, &one.State, &one.ByDefault)
		if err != nil {
			return nil, fmt.Errorf("read a destination a post may send to: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the destinations a post may send to: %w", err)
	}
	return found, nil
}
