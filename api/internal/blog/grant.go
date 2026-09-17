package blog

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type Holder struct {
	ID     uuid.UUID
	Handle string
}

type Grant struct {
	ID                    uuid.UUID
	Holder                Holder
	App                   App
	Categories            []Category
	DefaultCategory       Category
	Destinations          []Choice
	DestinationsInherited bool
	GrantedBy             *uuid.UUID
	GrantedAt             time.Time
	RevokedAt             *time.Time
	Active                bool
}

type GrantEdit struct {
	Handle            string
	AppID             uuid.UUID
	CategoryIDs       []uuid.UUID
	DefaultCategoryID uuid.UUID
}

type GrantUpdate struct {
	CategoryIDs       []uuid.UUID
	DefaultCategoryID *uuid.UUID
}

func (s *Service) Grants(ctx context.Context) ([]Grant, error) {
	return s.grantsWhere(ctx, `order by grant_row.active desc, grant_row.granted_at desc`)
}

func (s *Service) Workspace(ctx context.Context, accountID uuid.UUID) ([]Grant, error) {
	return s.grantsWhere(ctx, `
		where grant_row.user_id = $1 and grant_row.active and app.retired_at is null
		order by app.position, app.created_at
	`, accountID)
}

func (s *Service) CreateGrant(ctx context.Context, actor uuid.UUID, in GrantEdit) (Grant, error) {
	holder, err := s.verifiedAccountByHandle(ctx, in.Handle)
	if err != nil {
		return Grant{}, err
	}
	app, err := s.app(ctx, in.AppID)
	if err != nil {
		return Grant{}, err
	}
	if app.Retired {
		return Grant{}, FieldError{Field: "appId", Message: "That app has been retired."}
	}
	allowed, err := s.allowedCategories(ctx, in.CategoryIDs, in.DefaultCategoryID)
	if err != nil {
		return Grant{}, err
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Grant{}, fmt.Errorf("begin grant: %w", err)
	}
	defer tx.Rollback(ctx)
	var locked uuid.UUID
	err = tx.QueryRow(ctx, `select id from users where id = $1 for update`, holder.ID).Scan(&locked)
	if err != nil {
		return Grant{}, fmt.Errorf("lock the account being granted: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into publication_grants (id, user_id, app_id, default_category_id, granted_by)
		values ($1, $2, $3, $4, $5)
	`, id, holder.ID, app.ID, in.DefaultCategoryID, actor)
	if isUniqueViolation(err) {
		return Grant{}, ErrAlreadyGranted
	}
	if err != nil {
		return Grant{}, fmt.Errorf("create grant: %w", err)
	}
	if err := writeGrantCategories(ctx, tx, id, allowed); err != nil {
		return Grant{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "grant.created",
		AppID: &app.ID, GrantID: &id, SubjectID: &holder.ID,
	})
	if err != nil {
		return Grant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Grant{}, fmt.Errorf("commit grant: %w", err)
	}
	return s.grant(ctx, id)
}

func (s *Service) UpdateGrant(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in GrantUpdate,
) (Grant, error) {
	current, err := s.grant(ctx, id)
	if err != nil {
		return Grant{}, err
	}
	if !current.Active {
		return Grant{}, ErrGrantRevoked
	}
	categoryIDs := current.categoryIDs()
	if in.CategoryIDs != nil {
		categoryIDs = in.CategoryIDs
	}
	defaultID := current.DefaultCategory.ID
	if in.DefaultCategoryID != nil {
		defaultID = *in.DefaultCategoryID
	}
	allowed, err := s.allowedCategories(ctx, categoryIDs, defaultID)
	if err != nil {
		return Grant{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Grant{}, fmt.Errorf("begin grant update: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_grants set default_category_id = $2 where id = $1 and active
	`, id, defaultID)
	if err != nil {
		return Grant{}, fmt.Errorf("update grant: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		delete from publication_grant_categories where grant_id = $1
	`, id); err != nil {
		return Grant{}, fmt.Errorf("clear grant categories: %w", err)
	}
	if err := writeGrantCategories(ctx, tx, id, allowed); err != nil {
		return Grant{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "grant.updated",
		AppID: &current.App.ID, GrantID: &id, SubjectID: &current.Holder.ID,
	})
	if err != nil {
		return Grant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Grant{}, fmt.Errorf("commit grant update: %w", err)
	}
	return s.grant(ctx, id)
}

func (s *Service) RevokeGrant(ctx context.Context, actor uuid.UUID, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin grant revocation: %w", err)
	}
	defer tx.Rollback(ctx)
	var holderID, appID uuid.UUID
	err = tx.QueryRow(ctx, `
		update publication_grants
		   set active = false, revoked_at = now()
		 where id = $1 and active
		returning user_id, app_id
	`, id).Scan(&holderID, &appID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrGrantNotFound
	}
	if err != nil {
		return fmt.Errorf("revoke grant: %w", err)
	}
	if err := stopSchedulesUnder(ctx, tx, id); err != nil {
		return err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "grant.revoked",
		AppID: &appID, GrantID: &id, SubjectID: &holderID,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit grant revocation: %w", err)
	}
	return nil
}

func writeGrantCategories(ctx context.Context, tx pgx.Tx, id uuid.UUID, allowed []Category) error {
	for _, category := range allowed {
		_, err := tx.Exec(ctx, `
			insert into publication_grant_categories (grant_id, category_id) values ($1, $2)
		`, id, category.ID)
		if err != nil {
			return fmt.Errorf("allow a grant category: %w", err)
		}
	}
	return nil
}

func (s *Service) allowedCategories(
	ctx context.Context,
	ids []uuid.UUID,
	defaultID uuid.UUID,
) ([]Category, error) {
	if len(ids) == 0 {
		return nil, FieldError{
			Field:   "categoryIds",
			Message: "Allow the contributor at least one category.",
		}
	}
	allowed := make([]Category, 0, len(ids))
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		category, err := s.category(ctx, id)
		if err != nil {
			return nil, err
		}
		if category.Retired {
			return nil, FieldError{
				Field:   "categoryIds",
				Message: "A retired category cannot be allowed.",
			}
		}
		allowed = append(allowed, category)
	}
	if !seen[defaultID] {
		return nil, FieldError{
			Field:   "defaultCategoryId",
			Message: "The default has to be one of the allowed categories.",
		}
	}
	return allowed, nil
}

func (s *Service) verifiedAccountByHandle(ctx context.Context, handle string) (Holder, error) {
	var holder Holder
	var verified bool
	err := s.pool.QueryRow(ctx, `
		select id, username, email_verified_at is not null from users where username = $1
	`, handle).Scan(&holder.ID, &holder.Handle, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return Holder{}, ErrAccountNotFound
	}
	if err != nil {
		return Holder{}, fmt.Errorf("find the account being granted: %w", err)
	}
	if !verified {
		return Holder{}, ErrAccountUnverified
	}
	return holder, nil
}

func (s *Service) Grant(ctx context.Context, id uuid.UUID) (Grant, error) {
	return s.grant(ctx, id)
}

func (s *Service) grant(ctx context.Context, id uuid.UUID) (Grant, error) {
	found, err := s.grantsWhere(ctx, `where grant_row.id = $1`, id)
	if err != nil {
		return Grant{}, err
	}
	if len(found) == 0 {
		return Grant{}, ErrGrantNotFound
	}
	return found[0], nil
}

func (s *Service) grantsWhere(ctx context.Context, clause string, args ...any) ([]Grant, error) {
	rows, err := s.pool.Query(ctx, selectGrants+clause, args...)
	if err != nil {
		return nil, fmt.Errorf("read publication grants: %w", err)
	}
	found, err := collectGrants(rows)
	if err != nil {
		return nil, err
	}
	found, err = s.withAllowedCategories(ctx, found)
	if err != nil {
		return nil, err
	}
	return s.withAllowedDestinations(ctx, found)
}

func (s *Service) withAllowedDestinations(ctx context.Context, found []Grant) ([]Grant, error) {
	for index := range found {
		allowed, err := s.GrantChoices(ctx, found[index].ID)
		if err != nil {
			return nil, err
		}
		found[index].Destinations = allowed
		if found[index].App.Destinations, err = s.AppChoices(ctx, found[index].App.ID); err != nil {
			return nil, err
		}
	}
	return found, nil
}

func (s *Service) withAllowedCategories(ctx context.Context, found []Grant) ([]Grant, error) {
	if len(found) == 0 {
		return found, nil
	}
	ids := make([]uuid.UUID, 0, len(found))
	for _, one := range found {
		ids = append(ids, one.ID)
	}
	rows, err := s.pool.Query(ctx, `
		select allowed.grant_id, category.id, category.slug, category.label,
		       category.position, category.retired_at is not null
		  from publication_grant_categories allowed
		  join publication_categories category on category.id = allowed.category_id
		 where allowed.grant_id = any($1)
		 order by category.position, category.created_at
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("read allowed grant categories: %w", err)
	}
	defer rows.Close()
	byGrant := make(map[uuid.UUID][]Category, len(found))
	for rows.Next() {
		var grantID uuid.UUID
		var one Category
		err := rows.Scan(&grantID, &one.ID, &one.Slug, &one.Label, &one.Position, &one.Retired)
		if err != nil {
			return nil, fmt.Errorf("read an allowed grant category: %w", err)
		}
		byGrant[grantID] = append(byGrant[grantID], one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read allowed grant categories: %w", err)
	}
	for index := range found {
		found[index].Categories = byGrant[found[index].ID]
	}
	return found, nil
}

const selectGrants = `
	select grant_row.id, holder.id, holder.username,
	       app.id, app.slug, app.name, app.home_url, app.position,
	       app.retired_at is not null,
	       fallback.id, fallback.slug, fallback.label, fallback.position,
	       fallback.retired_at is not null, not grant_row.destinations_overridden,
	       grant_row.granted_by, grant_row.granted_at, grant_row.revoked_at, grant_row.active
	  from publication_grants grant_row
	  join users holder on holder.id = grant_row.user_id
	  join publication_apps app on app.id = grant_row.app_id
	  join publication_categories fallback on fallback.id = grant_row.default_category_id
	`

func collectGrants(rows pgx.Rows) ([]Grant, error) {
	defer rows.Close()
	found := make([]Grant, 0, 8)
	for rows.Next() {
		var one Grant
		err := rows.Scan(
			&one.ID, &one.Holder.ID, &one.Holder.Handle,
			&one.App.ID, &one.App.Slug, &one.App.Name, &one.App.Home, &one.App.Position,
			&one.App.Retired,
			&one.DefaultCategory.ID, &one.DefaultCategory.Slug, &one.DefaultCategory.Label,
			&one.DefaultCategory.Position, &one.DefaultCategory.Retired,
			&one.DestinationsInherited,
			&one.GrantedBy, &one.GrantedAt, &one.RevokedAt, &one.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("read a publication grant: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication grants: %w", err)
	}
	return found, nil
}

func (g Grant) categoryIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(g.Categories))
	for _, category := range g.Categories {
		ids = append(ids, category.ID)
	}
	return ids
}

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

const categoryLabelLimit = 48

type Category struct {
	ID       uuid.UUID
	Slug     string
	Label    string
	Position int
	Retired  bool
}

type CategoryUpdate struct {
	Label   *string
	Retired *bool
}

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, selectCategories+` order by category.position, category.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read publication categories: %w", err)
	}
	defer rows.Close()
	return collectCategories(rows)
}

func (s *Service) UpdateCategory(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in CategoryUpdate,
) (Category, error) {
	current, err := s.category(ctx, id)
	if err != nil {
		return Category{}, err
	}
	label := current.Label
	if in.Label != nil {
		label = *in.Label
	}
	checked, err := plainField("label", label, categoryLabelLimit)
	if err != nil {
		return Category{}, err
	}
	if checked == "" {
		return Category{}, FieldError{Field: "label", Message: "Give the category a label."}
	}
	retired := current.Retired
	if in.Retired != nil {
		retired = *in.Retired
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Category{}, fmt.Errorf("begin category update: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_categories
		   set label = $2,
		       retired_at = case when $3 then coalesce(retired_at, now()) else null end,
		       updated_at = now()
		 where id = $1
	`, id, checked, retired)
	if err != nil {
		return Category{}, fmt.Errorf("update publication category: %w", err)
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "category.updated", CategoryID: &id}); err != nil {
		return Category{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Category{}, fmt.Errorf("commit category update: %w", err)
	}
	return s.category(ctx, id)
}

func (s *Service) OrderCategories(
	ctx context.Context,
	actor uuid.UUID,
	ids []uuid.UUID,
) ([]Category, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin category order: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `select id from publication_categories for update`)
	if err != nil {
		return nil, fmt.Errorf("lock publication categories: %w", err)
	}
	present, err := collectIDs(rows)
	if err != nil {
		return nil, err
	}
	if !sameSet(present, ids) {
		return nil, ErrIncompleteOrder
	}
	for position, id := range ids {
		_, err := tx.Exec(ctx, `
			update publication_categories set position = $2, updated_at = now() where id = $1
		`, id, position)
		if err != nil {
			return nil, fmt.Errorf("order publication category: %w", err)
		}
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "category.ordered"}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit category order: %w", err)
	}
	return s.Categories(ctx)
}

func (s *Service) category(ctx context.Context, id uuid.UUID) (Category, error) {
	rows, err := s.pool.Query(ctx, selectCategories+` where category.id = $1`, id)
	if err != nil {
		return Category{}, fmt.Errorf("read publication category: %w", err)
	}
	defer rows.Close()
	found, err := collectCategories(rows)
	if err != nil {
		return Category{}, err
	}
	if len(found) == 0 {
		return Category{}, ErrCategoryNotFound
	}
	return found[0], nil
}

const selectCategories = `
	select category.id, category.slug, category.label, category.position,
	       category.retired_at is not null
	  from publication_categories category
`

func collectCategories(rows pgx.Rows) ([]Category, error) {
	found := make([]Category, 0, 4)
	for rows.Next() {
		var one Category
		err := rows.Scan(&one.ID, &one.Slug, &one.Label, &one.Position, &one.Retired)
		if err != nil {
			return nil, fmt.Errorf("read a publication category: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication categories: %w", err)
	}
	return found, nil
}

func (s *Service) WritableCategories(
	ctx context.Context,
	held []Grant,
	admin bool,
) ([]Category, error) {
	if admin {
		live, err := s.Categories(ctx)
		if err != nil {
			return nil, err
		}
		open := make([]Category, 0, len(live))
		for _, one := range live {
			if !one.Retired {
				open = append(open, one)
			}
		}
		return open, nil
	}
	open := make([]Category, 0, 4)
	seen := make(map[uuid.UUID]bool)
	for _, grant := range held {
		for _, one := range grant.Categories {
			if seen[one.ID] || one.Retired {
				continue
			}
			seen[one.ID] = true
			open = append(open, one)
		}
	}
	return open, nil
}
