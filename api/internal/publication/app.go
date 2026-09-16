package publication

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const addressLimit = 300

type App struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	Home         string
	Position     int
	Retired      bool
	Destinations []Choice
}

func (s *Service) Apps(ctx context.Context) ([]App, error) {
	rows, err := s.pool.Query(ctx, selectApps+` order by app.position, app.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read publication apps: %w", err)
	}
	defer rows.Close()
	found, err := collectApps(rows)
	if err != nil {
		return nil, err
	}
	return s.withAppDestinations(ctx, found)
}

func (s *Service) withAppDestinations(ctx context.Context, found []App) ([]App, error) {
	for index := range found {
		allowed, err := s.AppChoices(ctx, found[index].ID)
		if err != nil {
			return nil, err
		}
		found[index].Destinations = allowed
	}
	return found, nil
}

func (s *Service) App(ctx context.Context, id uuid.UUID) (App, error) {
	return s.app(ctx, id)
}

func (s *Service) app(ctx context.Context, id uuid.UUID) (App, error) {
	rows, err := s.pool.Query(ctx, selectApps+` where app.id = $1`, id)
	if err != nil {
		return App{}, fmt.Errorf("read publication app: %w", err)
	}
	defer rows.Close()
	found, err := collectApps(rows)
	if err != nil {
		return App{}, err
	}
	if len(found) == 0 {
		return App{}, ErrAppNotFound
	}
	found, err = s.withAppDestinations(ctx, found)
	if err != nil {
		return App{}, err
	}
	return found[0], nil
}

const selectApps = `
	select app.id, app.slug, app.name, app.home_url, app.position,
	       app.retired_at is not null
	  from publication_apps app
`

func collectApps(rows pgx.Rows) ([]App, error) {
	found := make([]App, 0, 8)
	for rows.Next() {
		var one App
		err := rows.Scan(
			&one.ID, &one.Slug, &one.Name, &one.Home, &one.Position,
			&one.Retired,
		)
		if err != nil {
			return nil, fmt.Errorf("read a publication app: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication apps: %w", err)
	}
	return found, nil
}

func (s *Service) NameableApps(
	ctx context.Context,
	held []Grant,
	admin bool,
) ([]App, error) {
	if admin {
		configured, err := s.Apps(ctx)
		if err != nil {
			return nil, err
		}
		live := make([]App, 0, len(configured))
		for _, one := range configured {
			if !one.Retired {
				live = append(live, one)
			}
		}
		return live, nil
	}
	named := make([]App, 0, len(held))
	seen := make(map[uuid.UUID]bool, len(held))
	for _, grant := range held {
		if seen[grant.App.ID] || grant.App.Retired {
			continue
		}
		seen[grant.App.ID] = true
		named = append(named, grant.App)
	}
	return named, nil
}
