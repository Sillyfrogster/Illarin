package publication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// snapshot is one person's public identity as a post will carry it for good.
type snapshot struct {
	AccountID    uuid.UUID
	Handle       string
	DisplayName  string
	ContactEmail string
	AvatarID     *uuid.UUID
	Positions    []byte
	Distinctions []byte
	AppID        *uuid.UUID
	AppSlug      *string
	AppName      *string
}

// takeSnapshot reads one person's approved public identity and the app their grant publishes for, inside the caller's transaction.
func takeSnapshot(
	ctx context.Context,
	tx pgx.Tx,
	accountID uuid.UUID,
	grantID *uuid.UUID,
) (snapshot, error) {
	taken := snapshot{AccountID: accountID}
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
	`, accountID).Scan(
		&taken.Handle, &restricted, &taken.DisplayName, &taken.ContactEmail, &taken.AvatarID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return snapshot{}, ErrAccountNotFound
	}
	if err != nil {
		return snapshot{}, fmt.Errorf("read the author's public identity: %w", err)
	}
	if restricted {
		taken.DisplayName, taken.ContactEmail, taken.AvatarID = "", "", nil
	}
	positions, distinctions := []string{}, []string{}
	if !restricted {
		positions, distinctions, err = shownNames(ctx, tx, accountID)
		if err != nil {
			return snapshot{}, err
		}
	}
	if taken.Positions, err = json.Marshal(positions); err != nil {
		return snapshot{}, fmt.Errorf("write the author's positions: %w", err)
	}
	if taken.Distinctions, err = json.Marshal(distinctions); err != nil {
		return snapshot{}, fmt.Errorf("write the author's distinctions: %w", err)
	}
	taken.AppID, taken.AppSlug, taken.AppName, err = grantApp(ctx, tx, grantID)
	if err != nil {
		return snapshot{}, err
	}
	return taken, nil
}

// captureByline copies one person's public identity onto a post once, leaving a post that already carries a byline alone.
func captureByline(
	ctx context.Context,
	tx pgx.Tx,
	postID, authorID uuid.UUID,
	grantID *uuid.UUID,
) error {
	taken, err := takeSnapshot(ctx, tx, authorID, grantID)
	if err != nil {
		return err
	}
	return writeByline(ctx, tx, postID, taken, `on conflict (post_id) do nothing`)
}

// replaceByline puts a corrected attribution on a post that already went public.
func replaceByline(
	ctx context.Context,
	tx pgx.Tx,
	postID, accountID uuid.UUID,
	grantID *uuid.UUID,
) error {
	taken, err := takeSnapshot(ctx, tx, accountID, grantID)
	if err != nil {
		return err
	}
	return writeByline(ctx, tx, postID, taken, `
		on conflict (post_id) do update
		   set account_id = excluded.account_id, handle = excluded.handle,
		       display_name = excluded.display_name, contact_email = excluded.contact_email,
		       avatar_media_id = excluded.avatar_media_id, positions = excluded.positions,
		       distinctions = excluded.distinctions, app_id = excluded.app_id,
		       app_slug = excluded.app_slug, app_name = excluded.app_name,
		       captured_at = now()`)
}

func writeByline(
	ctx context.Context,
	tx pgx.Tx,
	postID uuid.UUID,
	taken snapshot,
	whenHeld string,
) error {
	_, err := tx.Exec(ctx, `
		insert into post_bylines (post_id, account_id, handle, display_name, contact_email,
		                          avatar_media_id, positions, distinctions,
		                          app_id, app_slug, app_name)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`+whenHeld,
		postID, taken.AccountID, taken.Handle, taken.DisplayName, taken.ContactEmail,
		taken.AvatarID, taken.Positions, taken.Distinctions,
		taken.AppID, taken.AppSlug, taken.AppName)
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

const selectBylines = `
	select byline.post_id, byline.account_id, byline.handle, byline.display_name,
	       byline.contact_email, avatar.id, avatar.width, avatar.height,
	       byline.positions, byline.distinctions,
	       app.id, app.slug, app.name, app.home_url, app.position,
	       app.retired_at is not null
	  from post_bylines byline
	  left join profile_media avatar
	         on avatar.id = byline.avatar_media_id and avatar.blob_id is not null
	  left join publication_apps app on app.id = byline.app_id
	`

func readByline(ctx context.Context, pool queryRower, postID uuid.UUID) (Byline, error) {
	_, found, err := scanByline(pool.QueryRow(ctx,
		selectBylines+` where byline.post_id = $1`, postID))
	return found, err
}

// bylinesFor answers the attribution each of these posts carries, if any has one.
func bylinesFor(
	ctx context.Context,
	pool rowQuerier,
	ids []uuid.UUID,
) (map[uuid.UUID]Byline, error) {
	rows, err := pool.Query(ctx, selectBylines+` where byline.post_id = any($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("read the post bylines: %w", err)
	}
	defer rows.Close()
	held := make(map[uuid.UUID]Byline, len(ids))
	for rows.Next() {
		postID, one, err := scanByline(rows)
		if err != nil {
			return nil, err
		}
		held[postID] = one
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the post bylines: %w", err)
	}
	return held, nil
}

func scanByline(row rowScanner) (uuid.UUID, Byline, error) {
	var postID uuid.UUID
	var found Byline
	var positions, distinctions []byte
	var avatarID *uuid.UUID
	var width, height *int
	var appID *uuid.UUID
	var appSlug, appName, appHome *string
	var appPosition *int
	var appRetired *bool
	err := row.Scan(
		&postID, &found.AccountID, &found.Handle, &found.DisplayName, &found.ContactEmail,
		&avatarID, &width, &height, &positions, &distinctions,
		&appID, &appSlug, &appName, &appHome, &appPosition, &appRetired,
	)
	if err != nil {
		return uuid.Nil, Byline{}, fmt.Errorf("read the post byline: %w", err)
	}
	found.Avatar = scanPortrait(avatarID, width, height)
	if err := json.Unmarshal(positions, &found.Positions); err != nil {
		return uuid.Nil, Byline{}, fmt.Errorf("read the byline positions: %w", err)
	}
	if err := json.Unmarshal(distinctions, &found.Distinctions); err != nil {
		return uuid.Nil, Byline{}, fmt.Errorf("read the byline distinctions: %w", err)
	}
	if appID != nil {
		found.App = &App{
			ID: *appID, Slug: *appSlug, Name: *appName, Home: *appHome,
			Position: *appPosition, Retired: *appRetired,
		}
	}
	return postID, found, nil
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

// execer is the little an address reservation needs of whatever writes it.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// rowQuerier is the little a batched attribution read needs of a pool.
type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// rowScanner is what a single row and a row in a set have in common.
type rowScanner interface {
	Scan(into ...any) error
}
