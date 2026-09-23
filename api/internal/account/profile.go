package account

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	displayNameLimit = 48
	biographyLimit   = 400
	linkLabelLimit   = 32
	linkAddressLimit = 300
	profileLinkLimit = 6
)

var (
	ErrProfileMediaNotFound = errors.New("profile media does not exist")
	ErrPictureMissing       = errors.New("profile has no such picture")
)

// Picture names one of the two pictures a profile carries.
type Picture string

const (
	Avatar Picture = "avatar"
	Banner Picture = "banner"
)

func (p Picture) column() string {
	if p == Banner {
		return "banner_media_id"
	}
	return "avatar_media_id"
}

func (p Picture) size() string {
	if p == Banner {
		return "detail"
	}
	return "grid"
}

func PictureURL(picture Picture, mediaID uuid.UUID, version uint32) string {
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, picture.size(), version)
}

type ProfileLink struct {
	Label   string
	Address string
}

type ProfilePicture struct {
	MediaID          uuid.UUID
	Width            int
	Height           int
	ImageSizeVersion uint32
}

type PublicProfile struct {
	ID                             uuid.UUID
	Handle                         string
	DisplayName                    string
	Biography                      string
	ContactEmail                   string
	Avatar                         *ProfilePicture
	Banner                         *ProfilePicture
	Tint                           string
	Links                          []ProfileLink
	FeaturedWorkIDs                []uuid.UUID
	Works                          int
	ShowNSFWContributionsOnProfile bool
	Restricted                     bool
}

type ProfileEdit struct {
	DisplayName  string
	Biography    string
	ContactEmail string
	Links        []ProfileLink
}

func (s *Service) PublicProfile(ctx context.Context, handle string) (PublicProfile, error) {
	var found PublicProfile
	var avatar, banner storedPicture
	var tint *string
	err := s.pool.QueryRow(ctx, `
		select account.id, account.username, account.show_nsfw_contributions_on_profile,
		       restricted.user_id is not null,
		       coalesce(profile.display_name, ''), coalesce(profile.biography, ''),
		       coalesce(profile.contact_email, ''), profile.banner_tint,
		       avatar.id, avatar.width, avatar.height,
		       banner.id, banner.width, banner.height,
		       (select count(*) from works
		         where owner_id = account.id and lifecycle = 'published' and visibility = 'listed'
		           and deleted_at is null and taken_down_at is null),
		       array(select work_id from profile_featured_works
		              where user_id = account.id order by position)
		  from users account
		  left join restricted_profiles restricted on restricted.user_id = account.id
		  left join public_profiles profile on profile.user_id = account.id
		  left join profile_media avatar
		         on avatar.id = profile.avatar_media_id and avatar.blob_id is not null
		  left join profile_media banner
		         on banner.id = profile.banner_media_id and banner.blob_id is not null
		 where account.username = $1
	`, handle).Scan(
		&found.ID, &found.Handle, &found.ShowNSFWContributionsOnProfile, &found.Restricted,
		&found.DisplayName, &found.Biography, &found.ContactEmail, &tint,
		&avatar.id, &avatar.width, &avatar.height,
		&banner.id, &banner.width, &banner.height,
		&found.Works, &found.FeaturedWorkIDs,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrProfileNotFound
	}
	if err != nil {
		return PublicProfile{}, fmt.Errorf("read public profile: %w", err)
	}
	if found.Restricted {
		return PublicProfile{
			ID:                             found.ID,
			Handle:                         found.Handle,
			ShowNSFWContributionsOnProfile: found.ShowNSFWContributionsOnProfile,
			Links:                          []ProfileLink{},
			FeaturedWorkIDs:                []uuid.UUID{},
			Works:                          found.Works,
			Restricted:                     true,
		}, nil
	}
	found.Avatar = avatar.picture()
	found.Banner = banner.picture()
	if found.Banner != nil && tint != nil {
		found.Tint = *tint
	}
	links, err := s.profileLinks(ctx, found.ID)
	if err != nil {
		return PublicProfile{}, err
	}
	found.Links = links
	return found, nil
}

func (s *Service) SaveProfile(ctx context.Context, owner api.Account, in ProfileEdit) (PublicProfile, error) {
	edit, err := validateProfileEdit(in)
	if err != nil {
		return PublicProfile{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin profile save: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockProfileForEdit(ctx, tx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into public_profiles (user_id, display_name, biography, contact_email)
		values ($1, $2, $3, $4)
		on conflict (user_id) do update
		   set display_name = excluded.display_name,
		       biography = excluded.biography,
		       contact_email = excluded.contact_email,
		       updated_at = now()
	`, owner.ID, edit.DisplayName, edit.Biography, edit.ContactEmail)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("save profile fields: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from public_profile_links where user_id = $1`, owner.ID); err != nil {
		return PublicProfile{}, fmt.Errorf("clear profile links: %w", err)
	}
	for position, link := range edit.Links {
		_, err := tx.Exec(ctx, `
			insert into public_profile_links (user_id, position, label, url)
			values ($1, $2, $3, $4)
		`, owner.ID, position, link.Label, link.Address)
		if err != nil {
			return PublicProfile{}, fmt.Errorf("save profile link: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit profile save: %w", err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}

// SetPicture stores an uploaded avatar or banner, retires the one it replaces and records a banner's tint.
func (s *Service) SetPicture(ctx context.Context, owner api.Account, picture Picture, file io.Reader) (PublicProfile, error) {
	if err := s.refuseWhileRestricted(ctx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	stored, prepared, err := s.media.Accept(ctx, file)
	if err != nil {
		return PublicProfile{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin %s change: %w", picture, err)
	}
	defer tx.Rollback(ctx)
	if err := lockProfileForEdit(ctx, tx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	mediaID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into profile_media (id, user_id, blob_id, width, height)
		values ($1, $2, $3, $4, $5)
	`, mediaID, owner.ID, stored.ID, prepared.Width, prepared.Height)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("record %s: %w", picture, err)
	}
	if err := replacePicture(ctx, tx, owner.ID, picture, &mediaID); err != nil {
		return PublicProfile{}, err
	}
	if picture == Banner {
		if _, err := tx.Exec(ctx, `update public_profiles set banner_tint = $2 where user_id = $1`,
			owner.ID, nullable(prepared.Tint)); err != nil {
			return PublicProfile{}, fmt.Errorf("record the banner tint: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit %s change: %w", picture, err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}

func (s *Service) RemovePicture(ctx context.Context, owner api.Account, picture Picture) (PublicProfile, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin %s removal: %w", picture, err)
	}
	defer tx.Rollback(ctx)
	if err := lockProfileForEdit(ctx, tx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	var present uuid.UUID
	err = tx.QueryRow(ctx, `
		select `+picture.column()+` from public_profiles
		 where user_id = $1 and `+picture.column()+` is not null
		 for update
	`, owner.ID).Scan(&present)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrPictureMissing
	}
	if err != nil {
		return PublicProfile{}, fmt.Errorf("read %s: %w", picture, err)
	}
	if err := replacePicture(ctx, tx, owner.ID, picture, nil); err != nil {
		return PublicProfile{}, err
	}
	if picture == Banner {
		if _, err := tx.Exec(ctx, `update public_profiles set banner_tint = null where user_id = $1`, owner.ID); err != nil {
			return PublicProfile{}, fmt.Errorf("clear the banner tint: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit %s removal: %w", picture, err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}

func (s *Service) AvatarImageSize(
	ctx context.Context,
	mediaID uuid.UUID,
	size string,
	version uint32,
) (string, string, error) {
	if _, known := mediaproc.ImageSizeByName(size); !known || version != mediaproc.ImageSizeVersion {
		return "", "", ErrProfileMediaNotFound
	}
	var blobID uuid.UUID
	var digestBytes []byte
	err := s.pool.QueryRow(ctx, `
		select media.blob_id, blob.sha256
		  from profile_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
	`, mediaID).Scan(&blobID, &digestBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrProfileMediaNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("find profile media: %w", err)
	}
	if len(digestBytes) != sha256.Size {
		return "", "", fmt.Errorf("profile media blob has a %d-byte digest", len(digestBytes))
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	redirect, err := s.media.Serve(ctx, blobID, digest, size, version)
	if err != nil {
		return "", "", err
	}
	return redirect, s.media.ImageSizeMediaType(), nil
}

func replacePicture(ctx context.Context, tx pgx.Tx, ownerID uuid.UUID, picture Picture, mediaID *uuid.UUID) error {
	column := picture.column()
	var superseded *uuid.UUID
	err := tx.QueryRow(ctx, `select `+column+` from public_profiles where user_id = $1`, ownerID).Scan(&superseded)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read the %s being replaced: %w", picture, err)
	}
	_, err = tx.Exec(ctx, `
		insert into public_profiles (user_id, `+column+`)
		values ($1, $2)
		on conflict (user_id) do update
		   set `+column+` = excluded.`+column+`, updated_at = now()
	`, ownerID, mediaID)
	if err != nil {
		return fmt.Errorf("point the profile at its %s: %w", picture, err)
	}
	if superseded == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `delete from profile_media where id = $1`, *superseded); err != nil {
		return fmt.Errorf("drop the superseded %s: %w", picture, err)
	}
	return nil
}

// storedPicture is one profile_media row as the profile query reads it, absent when every column is null.
type storedPicture struct {
	id            *uuid.UUID
	width, height *int
}

func (p storedPicture) picture() *ProfilePicture {
	if p.id == nil || p.width == nil || p.height == nil {
		return nil
	}
	return &ProfilePicture{
		MediaID: *p.id, Width: *p.width, Height: *p.height, ImageSizeVersion: mediaproc.ImageSizeVersion,
	}
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func lockProfileForEdit(ctx context.Context, tx pgx.Tx, ownerID uuid.UUID) error {
	var locked uuid.UUID
	if err := tx.QueryRow(ctx, `select id from users where id = $1 for update`, ownerID).Scan(&locked); err != nil {
		return fmt.Errorf("lock the account editing its profile: %w", err)
	}
	var restricted bool
	err := tx.QueryRow(ctx, `
		select exists (select 1 from restricted_profiles where user_id = $1)
	`, ownerID).Scan(&restricted)
	if err != nil {
		return fmt.Errorf("read whether the profile is restricted: %w", err)
	}
	if restricted {
		return ErrProfileRestricted
	}
	return nil
}

func (s *Service) profileLinks(ctx context.Context, ownerID uuid.UUID) ([]ProfileLink, error) {
	rows, err := s.pool.Query(ctx, `
		select label, url from public_profile_links where user_id = $1 order by position
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("read profile links: %w", err)
	}
	defer rows.Close()
	links := make([]ProfileLink, 0, profileLinkLimit)
	for rows.Next() {
		var link ProfileLink
		if err := rows.Scan(&link.Label, &link.Address); err != nil {
			return nil, fmt.Errorf("read a profile link: %w", err)
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read profile links: %w", err)
	}
	return links, nil
}

func validateProfileEdit(in ProfileEdit) (ProfileEdit, error) {
	displayName, err := plainField("displayName", in.DisplayName, displayNameLimit, false)
	if err != nil {
		return ProfileEdit{}, err
	}
	biography, err := plainField("biography", in.Biography, biographyLimit, true)
	if err != nil {
		return ProfileEdit{}, err
	}
	contactEmail := strings.TrimSpace(in.ContactEmail)
	if contactEmail != "" {
		contactEmail, err = normalizeEmail(contactEmail)
		if err != nil {
			return ProfileEdit{}, FieldError{
				Field:   "contactEmail",
				Message: "Public contact needs to look like name@example.com.",
			}
		}
	}
	if len(in.Links) > profileLinkLimit {
		return ProfileEdit{}, FieldError{
			Field:   "links",
			Message: fmt.Sprintf("A profile can show up to %d links.", profileLinkLimit),
		}
	}
	links := make([]ProfileLink, 0, len(in.Links))
	for _, link := range in.Links {
		checked, err := validateProfileLink(link)
		if err != nil {
			return ProfileEdit{}, err
		}
		links = append(links, checked)
	}
	return ProfileEdit{
		DisplayName:  displayName,
		Biography:    biography,
		ContactEmail: contactEmail,
		Links:        links,
	}, nil
}

func validateProfileLink(link ProfileLink) (ProfileLink, error) {
	label, err := plainField("links", link.Label, linkLabelLimit, false)
	if err != nil {
		return ProfileLink{}, err
	}
	address := strings.TrimSpace(link.Address)
	if label == "" || address == "" {
		return ProfileLink{}, FieldError{
			Field:   "links",
			Message: "Every link needs a label and an address.",
		}
	}
	if len(address) > linkAddressLimit {
		return ProfileLink{}, FieldError{
			Field:   "links",
			Message: fmt.Sprintf("A link address can be up to %d characters.", linkAddressLimit),
		}
	}
	parsed, parseErr := url.Parse(address)
	if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ProfileLink{}, FieldError{
			Field:   "links",
			Message: "A link address needs to start with https:// and name a site.",
		}
	}
	return ProfileLink{Label: label, Address: address}, nil
}

func plainField(field, raw string, limit int, allowBreaks bool) (string, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	for _, char := range value {
		if char == '\n' && allowBreaks {
			continue
		}
		if unicode.IsControl(char) {
			return "", FieldError{
				Field:   field,
				Message: "Use plain text without control characters.",
			}
		}
	}
	if len([]rune(value)) > limit {
		return "", FieldError{
			Field:   field,
			Message: fmt.Sprintf("Keep this to %d characters or fewer.", limit),
		}
	}
	return value, nil
}

func (s *Service) refuseWhileRestricted(ctx context.Context, ownerID uuid.UUID) error {
	var restricted bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from restricted_profiles where user_id = $1)
	`, ownerID).Scan(&restricted)
	if err != nil {
		return fmt.Errorf("read whether the profile is restricted: %w", err)
	}
	if restricted {
		return ErrProfileRestricted
	}
	return nil
}
