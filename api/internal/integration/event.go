package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const EventUpdatePublished = "asset.update.published.v1"

var ErrUnlistedConsentRequired = errors.New(
	"announcing an unlisted asset sends its direct link, which needs explicit consent",
)

type sent struct {
	ID         uuid.UUID  `json:"id"`
	Type       string     `json:"type"`
	OccurredAt time.Time  `json:"occurredAt"`
	Asset      sentAsset  `json:"asset"`
	Update     sentUpdate `json:"update"`
}

type sentAsset struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"`
	Name string    `json:"name"`
	URL  string    `json:"url"`
}

type sentUpdate struct {
	ID             uuid.UUID `json:"id"`
	Number         int       `json:"number"`
	VersionLabel   string    `json:"versionLabel,omitempty"`
	Summary        string    `json:"summary"`
	RecordedAt     time.Time `json:"recordedAt"`
	ContentChanged bool      `json:"contentChanged"`
	HistoryURL     string    `json:"historyUrl"`
}

type announced struct {
	owner     uuid.UUID
	name      string
	kind      string
	discovery string
}

// Announce queues a published version for its chosen integrations
func (s *Service) Announce(
	ctx context.Context,
	tx pgx.Tx,
	published asset.Update,
	choice asset.UpdateAnnouncement,
) error {
	var held announced
	err := tx.QueryRow(ctx, `
		select owner_id, name, kind, discovery from assets where id = $1
	`, published.AssetID).Scan(&held.owner, &held.name, &held.kind, &held.discovery)
	if err != nil {
		return fmt.Errorf("read the asset being announced: %w", err)
	}
	unlisted := held.discovery == string(asset.DiscoveryUnlisted)
	picked, err := s.pick(ctx, tx, held.owner, published.AssetID, unlisted, choice)
	if err != nil || len(picked) == 0 {
		return err
	}
	eventID := uuid.New()
	occurred := s.now().UTC()
	body, err := json.Marshal(sent{
		ID: eventID, Type: EventUpdatePublished, OccurredAt: occurred,
		Asset: sentAsset{
			ID: published.AssetID, Kind: held.kind, Name: held.name,
			URL: s.assetAddress(published.AssetID),
		},
		Update: sentUpdate{
			ID: published.ID, Number: published.Number, VersionLabel: published.VersionLabel,
			Summary: published.Summary, RecordedAt: published.RecordedAt,
			ContentChanged: published.ContentChanged,
			HistoryURL:     s.historyAddress(published.AssetID, published.Number),
		},
	})
	if err != nil {
		return fmt.Errorf("write the update announcement: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into asset_update_events
		       (id, asset_id, snapshot_id, type, occurred_at, unlisted_consent, payload)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, eventID, published.AssetID, published.ID, EventUpdatePublished, occurred,
		choice.AnnounceUnlisted, body)
	if err != nil {
		return fmt.Errorf("record the update announcement: %w", err)
	}
	for _, one := range picked {
		_, err := tx.Exec(ctx, `
			insert into asset_update_deliveries
			       (id, event_id, destination_id, destination_name, destination_kind, due_at)
			values ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), eventID, one.ID, one.Name, one.Kind, occurred)
		if err != nil {
			return fmt.Errorf("keep the announcement work: %w", err)
		}
	}
	return nil
}

type picked struct {
	ID   uuid.UUID
	Name string
	Kind string
}

func (s *Service) pick(
	ctx context.Context,
	tx pgx.Tx,
	owner, assetID uuid.UUID,
	unlisted bool,
	choice asset.UpdateAnnouncement,
) ([]picked, error) {
	if choice.DestinationIDs == nil {
		if unlisted {
			return nil, nil
		}
		return collectPicked(tx.Query(ctx, `
			select destination.id, destination.name, destination.kind
			  from asset_update_destination_defaults chosen
			  join asset_update_destinations destination on destination.id = chosen.destination_id
			 where chosen.asset_id = $1 and destination.owner_id = $2 and destination.state = $3
			 order by destination.name, destination.id
			 for share of destination
		`, assetID, owner, Active))
	}
	wanted := distinct(*choice.DestinationIDs)
	if len(wanted) > 0 && unlisted && !choice.AnnounceUnlisted {
		return nil, ErrUnlistedConsentRequired
	}
	found, err := collectPicked(tx.Query(ctx, `
		select id, name, kind from asset_update_destinations
		 where owner_id = $1 and id = any($2::uuid[]) and state = $3
		 order by name, id
		 for share
	`, owner, wanted, Active))
	if err != nil {
		return nil, err
	}
	if len(found) != len(wanted) {
		return nil, asset.ErrUpdateDestinationIneligible
	}
	if err := remember(ctx, tx, assetID, wanted); err != nil {
		return nil, err
	}
	return found, nil
}

func remember(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, ids []uuid.UUID) error {
	_, err := tx.Exec(ctx, `delete from asset_update_destination_defaults where asset_id = $1`, assetID)
	if err != nil {
		return fmt.Errorf("forget the remembered destinations: %w", err)
	}
	for _, id := range ids {
		_, err := tx.Exec(ctx, `
			insert into asset_update_destination_defaults (asset_id, destination_id) values ($1, $2)
		`, assetID, id)
		if err != nil {
			return fmt.Errorf("remember a destination: %w", err)
		}
	}
	return nil
}

func distinct(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	kept := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		kept = append(kept, id)
	}
	return kept
}

func collectPicked(rows pgx.Rows, err error) ([]picked, error) {
	if err != nil {
		return nil, fmt.Errorf("read the destinations an update goes to: %w", err)
	}
	defer rows.Close()
	found := make([]picked, 0, 4)
	for rows.Next() {
		var one picked
		if err := rows.Scan(&one.ID, &one.Name, &one.Kind); err != nil {
			return nil, fmt.Errorf("read one destination an update goes to: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the destinations an update goes to: %w", err)
	}
	return found, nil
}

func (s *Service) assetAddress(assetID uuid.UUID) string {
	return s.site + "/a/" + assetID.String()
}

func (s *Service) historyAddress(assetID uuid.UUID, number int) string {
	return s.assetAddress(assetID) + "/history#version-" + strconv.Itoa(number)
}
