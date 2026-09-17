package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	Webhook    = "webhook"
	Discord    = "discord"
	Active     = "active"
	Disabled   = "disabled"
	Unverified = "unverified"
)

var ErrNotFound = errors.New("no such asset update destination")

type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string { return e.Message }

type Sender interface {
	Check(string) (string, error)
	Get(context.Context, string) (dispatch.Answer, error)
	Post(context.Context, string, map[string]string, []byte) (dispatch.Answer, error)
}

type Service struct {
	pool    *pgxpool.Pool
	sealing secrets.Key
	sender  Sender
	site    string
	ledger  dispatch.Ledger
	now     func() time.Time
}

var deliveryTables = dispatch.Tables{
	Deliveries: "asset_update_deliveries", Attempts: "asset_update_delivery_attempts",
}

func NewService(pool *pgxpool.Pool, sealing secrets.Key, sender Sender, site string) *Service {
	return &Service{
		pool: pool, sealing: sealing, sender: sender, site: strings.TrimRight(site, "/"),
		ledger: dispatch.NewLedger(pool, deliveryTables), now: time.Now,
	}
}

type Destination struct {
	version             int64
	ID                  uuid.UUID
	Kind                string
	Name                string
	Host                string
	Address             string
	State               string
	GuildID             *string
	ChannelID           *string
	SecretSetAt         *time.Time
	PreviousSecretUntil *time.Time
	VerifiedAt          *time.Time
	DisabledAt          *time.Time
	CreatedAt           time.Time
}

type Added struct {
	Destination Destination
	Secret      string
}

func (s *Service) Add(ctx context.Context, owner uuid.UUID, kind, name, address string) (Added, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 48 {
		return Added{}, FieldError{"name", "Name the destination in 1 to 48 characters."}
	}
	if kind != Webhook && kind != Discord {
		return Added{}, FieldError{"kind", "Choose a webhook or Discord destination."}
	}
	prepared, err := s.prepareAddress(ctx, kind, address)
	if err != nil {
		return Added{}, err
	}
	var secret string
	var sealedSecret []byte
	var secretAt *time.Time
	if kind == Webhook {
		secret, err = dispatch.MintSecret()
		if err != nil {
			return Added{}, err
		}
		sealedSecret, err = s.sealing.Seal([]byte(secret))
		if err != nil {
			return Added{}, err
		}
		now := time.Now().UTC()
		secretAt = &now
	}
	id := uuid.New()
	_, err = s.pool.Exec(ctx, `
		insert into asset_update_destinations
		    (id, owner_id, kind, name, host, address, signing_secret, signing_secret_set_at,
		     state, guild_id, channel_id, verified_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, id, owner, kind, name, prepared.host, prepared.sealed, sealedSecret, secretAt,
		prepared.state, prepared.guildID, prepared.channelID, prepared.verifiedAt)
	if err != nil {
		return Added{}, fmt.Errorf("create asset update destination: %w", err)
	}
	found, err := s.Get(ctx, owner, id)
	return Added{Destination: found, Secret: secret}, err
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Destination, error) {
	rows, err := s.pool.Query(ctx, selectDestinations+` where owner_id = $1 order by name, id`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := make([]Destination, 0)
	for rows.Next() {
		one, err := readDestination(rows)
		if err != nil {
			return nil, err
		}
		found = append(found, one)
	}
	return found, rows.Err()
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (Destination, error) {
	return readDestination(s.pool.QueryRow(ctx, selectDestinations+` where owner_id = $1 and id = $2`, owner, id))
}

const selectDestinations = `
	select id, kind, name, host, state, guild_id, channel_id, signing_secret_set_at,
	       previous_secret_until, verified_at, disabled_at, created_at, version
	  from asset_update_destinations
`

func readDestination(row pgx.Row) (Destination, error) {
	var d Destination
	err := row.Scan(&d.ID, &d.Kind, &d.Name, &d.Host, &d.State, &d.GuildID, &d.ChannelID,
		&d.SecretSetAt, &d.PreviousSecretUntil, &d.VerifiedAt, &d.DisabledAt, &d.CreatedAt, &d.version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Destination{}, ErrNotFound
	}
	d.Address = "https://" + d.Host + "/…"
	return d, err
}
