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

// avatarVariant is the one size a profile avatar is published at.
const avatarVariant = "grid"

var (
	ErrProfileMediaNotFound = errors.New("profile media does not exist")
	ErrAvatarMissing        = errors.New("profile has no avatar")
)

// AvatarURL addresses one avatar on the byte path every image shares.
func AvatarURL(mediaID uuid.UUID, version uint32) string {
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, avatarVariant, version)
}

// ProfileLink is one labelled address a creator chose to publish.
type ProfileLink struct {
	Label   string
	Address string
}

// ProfileAvatar is the creator's uploaded portrait.
type ProfileAvatar struct {
	MediaID           uuid.UUID
	Width             int
	Height            int
	DerivativeVersion uint32
}

// PublicProfile is everything a visitor may see about an account.
type PublicProfile struct {
	ID                             uuid.UUID
	Handle                         string
	DisplayName                    string
	Biography                      string
	ContactEmail                   string
	Avatar                         *ProfileAvatar
	Links                          []ProfileLink
	ShowNSFWContributionsOnProfile bool
}

// ProfileEdit is the whole set of added fields. An empty value removes one.
type ProfileEdit struct {
	DisplayName  string
	Biography    string
	ContactEmail string
	Links        []ProfileLink
}

// PublicProfile answers the profile behind a handle.
func (s *Service) PublicProfile(ctx context.Context, handle string) (PublicProfile, error) {
	var found PublicProfile
	var avatarID *uuid.UUID
	var width, height *int
	err := s.pool.QueryRow(ctx, `
		select account.id, account.username, account.show_nsfw_contributions_on_profile,
		       coalesce(profile.display_name, ''), coalesce(profile.biography, ''),
		       coalesce(profile.contact_email, ''),
		       avatar.id, avatar.width, avatar.height
		  from users account
		  left join public_profiles profile on profile.user_id = account.id
		  left join profile_media avatar
		         on avatar.id = profile.avatar_media_id and avatar.blob_id is not null
		 where account.username = $1
	`, handle).Scan(
		&found.ID, &found.Handle, &found.ShowNSFWContributionsOnProfile,
		&found.DisplayName, &found.Biography, &found.ContactEmail,
		&avatarID, &width, &height,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrProfileNotFound
	}
	if err != nil {
		return PublicProfile{}, fmt.Errorf("read public profile: %w", err)
	}
	if avatarID != nil && width != nil && height != nil {
		found.Avatar = &ProfileAvatar{
			MediaID:           *avatarID,
			Width:             *width,
			Height:            *height,
			DerivativeVersion: mediaproc.DerivativeVersion,
		}
	}
	links, err := s.profileLinks(ctx, found.ID)
	if err != nil {
		return PublicProfile{}, err
	}
	found.Links = links
	return found, nil
}

// SaveProfile replaces every added field on one account's profile.
func (s *Service) SaveProfile(ctx context.Context, owner Account, in ProfileEdit) (PublicProfile, error) {
	edit, err := validateProfileEdit(in)
	if err != nil {
		return PublicProfile{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin profile save: %w", err)
	}
	defer tx.Rollback(ctx)
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

// SetAvatar puts an image at a fresh address and drops the one it replaces.
func (s *Service) SetAvatar(ctx context.Context, owner Account, file io.Reader) (PublicProfile, error) {
	stored, prepared, err := s.media.Accept(ctx, file)
	if err != nil {
		return PublicProfile{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin avatar change: %w", err)
	}
	defer tx.Rollback(ctx)
	mediaID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into profile_media (id, user_id, blob_id, width, height)
		values ($1, $2, $3, $4, $5)
	`, mediaID, owner.ID, stored.ID, prepared.Width, prepared.Height)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("record avatar: %w", err)
	}
	if err := replaceAvatar(ctx, tx, owner.ID, &mediaID); err != nil {
		return PublicProfile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit avatar change: %w", err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}

// RemoveAvatar takes the portrait off a profile and lets its bytes go.
func (s *Service) RemoveAvatar(ctx context.Context, owner Account) (PublicProfile, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin avatar removal: %w", err)
	}
	defer tx.Rollback(ctx)
	var present uuid.UUID
	err = tx.QueryRow(ctx, `
		select avatar_media_id from public_profiles
		 where user_id = $1 and avatar_media_id is not null
		 for update
	`, owner.ID).Scan(&present)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrAvatarMissing
	}
	if err != nil {
		return PublicProfile{}, fmt.Errorf("read avatar: %w", err)
	}
	if err := replaceAvatar(ctx, tx, owner.ID, nil); err != nil {
		return PublicProfile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit avatar removal: %w", err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}

// AvatarVariant serves one size of a profile avatar. Every avatar is public.
func (s *Service) AvatarVariant(
	ctx context.Context,
	mediaID uuid.UUID,
	variant string,
	version uint32,
) (string, string, error) {
	if _, known := mediaproc.VariantByName(variant); !known || version != mediaproc.DerivativeVersion {
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
	redirect, err := s.media.Serve(ctx, blobID, digest, variant, version)
	if err != nil {
		return "", "", err
	}
	return redirect, s.media.DerivativeType(), nil
}

func replaceAvatar(ctx context.Context, tx pgx.Tx, ownerID uuid.UUID, mediaID *uuid.UUID) error {
	var superseded *uuid.UUID
	err := tx.QueryRow(ctx, `
		select avatar_media_id from public_profiles where user_id = $1 for update
	`, ownerID).Scan(&superseded)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read the avatar being replaced: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into public_profiles (user_id, avatar_media_id)
		values ($1, $2)
		on conflict (user_id) do update
		   set avatar_media_id = excluded.avatar_media_id, updated_at = now()
	`, ownerID, mediaID)
	if err != nil {
		return fmt.Errorf("point the profile at its avatar: %w", err)
	}
	if superseded == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `delete from profile_media where id = $1`, *superseded); err != nil {
		return fmt.Errorf("drop the superseded avatar: %w", err)
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
