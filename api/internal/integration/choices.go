package integration

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type IntegrationChoice struct {
	ID        uuid.UUID
	Name      string
	Type      string
	ByDefault bool
}

func (s *Service) Integrations(ctx context.Context, ownerID, workID uuid.UUID) ([]IntegrationChoice, error) {
	var owned bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from works where id = $1 and owner_id = $2 and deleted_at is null)
	`, workID, ownerID).Scan(&owned)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, work.ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		select integration.id, integration.name, integration.type,
		       exists (select 1 from work_integration_defaults chosen
		                where chosen.work_id = $2 and chosen.integration_id = integration.id)
		  from work_integrations integration
		 where integration.owner_id = $1 and integration.state = 'active'
		 order by integration.name, integration.id
	`, ownerID, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	choices := make([]IntegrationChoice, 0)
	for rows.Next() {
		var one IntegrationChoice
		if err := rows.Scan(&one.ID, &one.Name, &one.Type, &one.ByDefault); err != nil {
			return nil, err
		}
		choices = append(choices, one)
	}
	return choices, rows.Err()
}

func (s *Service) SetIntegrations(ctx context.Context, ownerID, workID uuid.UUID, ids []uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		select id from work_integrations
		 where owner_id = $1 and id = any($2::uuid[]) and state = 'active'
		 order by id for share
	`, ownerID, ids)
	if err != nil {
		return err
	}
	count := 0
	for rows.Next() {
		count++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if count != len(ids) {
		return version.ErrIntegrationIneligible
	}
	if _, err := tx.Exec(ctx, `delete from work_integration_defaults where work_id = $1`, workID); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `insert into work_integration_defaults (work_id, integration_id) values ($1, $2)`, workID, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
