package publication

import (
	"context"
	"encoding/json"
	"fmt"

	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Byline is the public attribution one post carries. It is copied when the post
// first goes public and is not rewritten by later profile changes.
//
// The avatar is held as the media record the profile wore at the time. Replacing
// an avatar deletes the record it replaced, so an old byline falls back to the
// handle rather than showing a portrait the person has since changed.
type Byline struct {
	AccountID    *uuid.UUID
	Handle       string
	DisplayName  string
	ContactEmail string
	Avatar       *Portrait
	Positions    []string
	Distinctions []string
	App          *App
}

// Portrait is the avatar a byline wore when its post went public.
type Portrait struct {
	MediaID           uuid.UUID
	Width             int
	Height            int
	DerivativeVersion uint32
}

// captureByline copies one person's public identity onto a post, once. The app
// is read from the grant inside the same transaction, so a revocation racing
// with publication cannot leave an attribution the grant no longer supports.
func captureByline(
	ctx context.Context,
	tx pgx.Tx,
	postID, authorID uuid.UUID,
	grantID *uuid.UUID,
) error {
	var handle, displayName, contactEmail string
	var avatarID *uuid.UUID
	var restricted bool
	err := tx.QueryRow(ctx, `
		select account.username, restriction.user_id is not null,
		       coalesce(profile.display_name, ''), coalesce(profile.contact_email, ''),
		       avatar.id
		  from users account
		  left join profile_restrictions restriction on restriction.user_id = account.id
		  left join public_profiles profile on profile.user_id = account.id
		  left join profile_media avatar
		         on avatar.id = profile.avatar_media_id and avatar.blob_id is not null
		 where account.id = $1
	`, authorID).Scan(&handle, &restricted, &displayName, &contactEmail, &avatarID)
	if err != nil {
		return fmt.Errorf("read the author's public identity: %w", err)
	}
	if restricted {
		displayName, contactEmail, avatarID = "", "", nil
	}
	positions, distinctions := []string{}, []string{}
	if !restricted {
		positions, distinctions, err = shownNames(ctx, tx, authorID)
		if err != nil {
			return err
		}
	}
	shownPositions, err := json.Marshal(positions)
	if err != nil {
		return fmt.Errorf("write the author's positions: %w", err)
	}
	shownDistinctions, err := json.Marshal(distinctions)
	if err != nil {
		return fmt.Errorf("write the author's distinctions: %w", err)
	}
	appID, appSlug, appName, err := grantApp(ctx, tx, grantID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into post_bylines (post_id, account_id, handle, display_name, contact_email,
		                          avatar_media_id, positions, distinctions,
		                          app_id, app_slug, app_name)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		on conflict (post_id) do nothing
	`, postID, authorID, handle, displayName, contactEmail, avatarID,
		shownPositions, shownDistinctions, appID, appSlug, appName)
	if err != nil {
		return fmt.Errorf("record the post byline: %w", err)
	}
	return nil
}

// grantApp answers the app a grant publishes for, or nothing for an Illarin post.
func grantApp(
	ctx context.Context,
	tx pgx.Tx,
	grantID *uuid.UUID,
) (*uuid.UUID, *string, *string, error) {
	if grantID == nil {
		return nil, nil, nil, nil
	}
	var id uuid.UUID
	var slug, name string
	err := tx.QueryRow(ctx, `
		select app.id, app.slug, app.name
		  from publication_grants grant_row
		  join publication_apps app on app.id = grant_row.app_id
		 where grant_row.id = $1
	`, *grantID).Scan(&id, &slug, &name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read the app a post publishes for: %w", err)
	}
	return &id, &slug, &name, nil
}

// shownNames answers the positions and the bounded showcase a visitor sees.
func shownNames(ctx context.Context, tx pgx.Tx, accountID uuid.UUID) ([]string, []string, error) {
	rows, err := tx.Query(ctx, `
		select definition.form, definition.name
		  from profile_distinction_assignments assignment
		  join profile_distinctions definition on definition.id = assignment.distinction_id
		 where assignment.user_id = $1 and assignment.active and definition.retired_at is null
		 order by case when definition.form = 'position' then 0 else 1 end,
		          case when definition.form = 'position' then definition.position else assignment.position end,
		          assignment.assigned_at
	`, accountID)
	if err != nil {
		return nil, nil, fmt.Errorf("read the author's distinctions: %w", err)
	}
	defer rows.Close()
	positions, distinctions := []string{}, []string{}
	for rows.Next() {
		var form, name string
		if err := rows.Scan(&form, &name); err != nil {
			return nil, nil, fmt.Errorf("read one of the author's distinctions: %w", err)
		}
		if Form(form) == FormPosition {
			positions = append(positions, name)
			continue
		}
		if len(distinctions) < showcaseLimit {
			distinctions = append(distinctions, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("read the author's distinctions: %w", err)
	}
	return positions, distinctions, nil
}

func readByline(ctx context.Context, pool queryRower, postID uuid.UUID) (Byline, error) {
	var found Byline
	var positions, distinctions []byte
	var avatarID *uuid.UUID
	var width, height *int
	var appID *uuid.UUID
	var appSlug, appName, appHome *string
	var appPosition *int
	var appRetired *bool
	err := pool.QueryRow(ctx, `
		select byline.account_id, byline.handle, byline.display_name, byline.contact_email,
		       avatar.id, avatar.width, avatar.height,
		       byline.positions, byline.distinctions,
		       app.id, app.slug, app.name, app.home_url, app.position,
		       app.retired_at is not null
		  from post_bylines byline
		  left join profile_media avatar
		         on avatar.id = byline.avatar_media_id and avatar.blob_id is not null
		  left join publication_apps app on app.id = byline.app_id
		 where byline.post_id = $1
	`, postID).Scan(
		&found.AccountID, &found.Handle, &found.DisplayName, &found.ContactEmail,
		&avatarID, &width, &height, &positions, &distinctions,
		&appID, &appSlug, &appName, &appHome, &appPosition, &appRetired,
	)
	if err != nil {
		return Byline{}, fmt.Errorf("read the post byline: %w", err)
	}
	found.Avatar = scanPortrait(avatarID, width, height)
	if err := json.Unmarshal(positions, &found.Positions); err != nil {
		return Byline{}, fmt.Errorf("read the byline positions: %w", err)
	}
	if err := json.Unmarshal(distinctions, &found.Distinctions); err != nil {
		return Byline{}, fmt.Errorf("read the byline distinctions: %w", err)
	}
	if appID != nil {
		found.App = &App{
			ID: *appID, Slug: *appSlug, Name: *appName, Home: *appHome,
			Position: *appPosition, Retired: *appRetired,
		}
	}
	return found, nil
}

func scanPortrait(mediaID *uuid.UUID, width, height *int) *Portrait {
	if mediaID == nil || width == nil || height == nil {
		return nil
	}
	return &Portrait{
		MediaID:           *mediaID,
		Width:             *width,
		Height:            *height,
		DerivativeVersion: mediaproc.DerivativeVersion,
	}
}

// queryRower is the little a byline read needs, so it works on a pool or inside
// the transaction that just wrote one.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
