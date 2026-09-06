package publication

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	appNameLimit = 48
	slugLimit    = 40
	addressLimit = 300
)

// slugPattern is the shape every public publication slug has to take.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// App is one project the publication carries posts for.
type App struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	Home         string
	Mark         *Mark
	Position     int
	Retired      bool
	Destinations []Choice
}

// AppEdit is the app text the authority supplies.
type AppEdit struct {
	Slug string
	Name string
	Home string
}

// AppUpdate carries only the parts of an app a request named.
type AppUpdate struct {
	Slug    *string
	Name    *string
	Home    *string
	Retired *bool
}

// Apps answers every configured app, retired ones included.
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

// withAppDestinations gives each app the destinations it allows, which is what
// every grant on it follows unless the grant names its own.
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

// DefineApp records a new app below the ones already configured.
func (s *Service) DefineApp(ctx context.Context, actor uuid.UUID, in AppEdit) (App, error) {
	edit, err := validateApp(in)
	if err != nil {
		return App{}, err
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return App{}, fmt.Errorf("begin app definition: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into publication_apps (id, slug, name, home_url, position)
		values ($1, $2, $3, $4, coalesce((select max(position) + 1 from publication_apps), 0))
	`, id, edit.Slug, edit.Name, edit.Home)
	if isUniqueViolation(err) {
		return App{}, ErrSlugTaken
	}
	if err != nil {
		return App{}, fmt.Errorf("define publication app: %w", err)
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "app.defined", AppID: &id}); err != nil {
		return App{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return App{}, fmt.Errorf("commit app definition: %w", err)
	}
	return s.app(ctx, id)
}

// UpdateApp changes what one app says and whether it is still current.
func (s *Service) UpdateApp(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in AppUpdate,
) (App, error) {
	current, err := s.app(ctx, id)
	if err != nil {
		return App{}, err
	}
	edit := AppEdit{Slug: current.Slug, Name: current.Name, Home: current.Home}
	if in.Slug != nil {
		edit.Slug = *in.Slug
	}
	if in.Name != nil {
		edit.Name = *in.Name
	}
	if in.Home != nil {
		edit.Home = *in.Home
	}
	checked, err := validateApp(edit)
	if err != nil {
		return App{}, err
	}
	retired := current.Retired
	if in.Retired != nil {
		retired = *in.Retired
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return App{}, fmt.Errorf("begin app update: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_apps
		   set slug = $2,
		       name = $3,
		       home_url = $4,
		       retired_at = case when $5 then coalesce(retired_at, now()) else null end,
		       updated_at = now()
		 where id = $1
	`, id, checked.Slug, checked.Name, checked.Home, retired)
	if isUniqueViolation(err) {
		return App{}, ErrSlugTaken
	}
	if err != nil {
		return App{}, fmt.Errorf("update publication app: %w", err)
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "app.updated", AppID: &id}); err != nil {
		return App{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return App{}, fmt.Errorf("commit app update: %w", err)
	}
	return s.app(ctx, id)
}

// OrderApps puts the configured apps in the order it is given them.
func (s *Service) OrderApps(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) ([]App, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin app order: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `select id from publication_apps for update`)
	if err != nil {
		return nil, fmt.Errorf("lock publication apps: %w", err)
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
			update publication_apps set position = $2, updated_at = now() where id = $1
		`, id, position)
		if err != nil {
			return nil, fmt.Errorf("order publication app: %w", err)
		}
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "app.ordered"}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit app order: %w", err)
	}
	return s.Apps(ctx)
}

// SetAppMark puts an Illarin-hosted image on an app and drops the one it replaces.
func (s *Service) SetAppMark(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	file io.Reader,
) (App, error) {
	if _, err := s.app(ctx, id); err != nil {
		return App{}, err
	}
	stored, prepared, err := s.media.Accept(ctx, file)
	if err != nil {
		return App{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return App{}, fmt.Errorf("begin app mark change: %w", err)
	}
	defer tx.Rollback(ctx)
	err = s.replaceMark(ctx, tx, appMarks, id, stored.ID, prepared.Width, prepared.Height)
	if err != nil {
		return App{}, err
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "app.marked", AppID: &id}); err != nil {
		return App{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return App{}, fmt.Errorf("commit app mark change: %w", err)
	}
	return s.app(ctx, id)
}

// App answers one configured app and the destinations it allows.
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
	       app.retired_at is not null, mark.id, mark.width, mark.height
	  from publication_apps app
	  left join publication_media mark
	         on mark.id = app.mark_media_id and mark.blob_id is not null
`

func collectApps(rows pgx.Rows) ([]App, error) {
	found := make([]App, 0, 8)
	for rows.Next() {
		var one App
		var markID *uuid.UUID
		var width, height *int
		err := rows.Scan(
			&one.ID, &one.Slug, &one.Name, &one.Home, &one.Position,
			&one.Retired, &markID, &width, &height,
		)
		if err != nil {
			return nil, fmt.Errorf("read a publication app: %w", err)
		}
		one.Mark = scanMark(markID, width, height)
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication apps: %w", err)
	}
	return found, nil
}

func validateApp(in AppEdit) (AppEdit, error) {
	slug, err := validateSlug(in.Slug)
	if err != nil {
		return AppEdit{}, err
	}
	name, err := plainField("name", in.Name, appNameLimit)
	if err != nil {
		return AppEdit{}, err
	}
	if name == "" {
		return AppEdit{}, FieldError{Field: "name", Message: "Give the app a public name."}
	}
	home, err := plainField("home", in.Home, addressLimit)
	if err != nil {
		return AppEdit{}, err
	}
	if !strings.HasPrefix(home, "https://") {
		return AppEdit{}, FieldError{
			Field:   "home",
			Message: "The app's address has to start with https://.",
		}
	}
	return AppEdit{Slug: slug, Name: name, Home: home}, nil
}

func validateSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if len([]rune(slug)) > slugLimit {
		return "", FieldError{
			Field:   "slug",
			Message: fmt.Sprintf("Keep the slug to %d characters or fewer.", slugLimit),
		}
	}
	if !slugPattern.MatchString(slug) {
		return "", FieldError{
			Field:   "slug",
			Message: "A slug is lower-case letters, digits and single hyphens.",
		}
	}
	return slug, nil
}

// NameableApps answers the publication apps one account may name in a release.
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
