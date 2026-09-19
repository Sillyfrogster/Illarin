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

var ErrNotFound = errors.New("no such work update integration")

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

var attemptTables = dispatch.Tables{
	Attempts: "work_announcement_attempts", Tries: "work_announcement_tries",
}

func NewService(pool *pgxpool.Pool, sealing secrets.Key, sender Sender, site string) *Service {
	return &Service{
		pool: pool, sealing: sealing, sender: sender, site: strings.TrimRight(site, "/"),
		ledger: dispatch.NewLedger(pool, attemptTables), now: time.Now,
	}
}

type Integration struct {
	version             int64
	ID                  uuid.UUID
	Type                string
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
	Integration Integration
	Secret      string
}

func (s *Service) Add(ctx context.Context, owner uuid.UUID, integrationType, name, address string) (Added, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 48 {
		return Added{}, FieldError{"name", "Name the integration in 1 to 48 characters."}
	}
	if integrationType != Webhook && integrationType != Discord {
		return Added{}, FieldError{"type", "Choose a webhook or Discord integration."}
	}
	prepared, err := s.prepareAddress(ctx, integrationType, address)
	if err != nil {
		return Added{}, err
	}
	var secret string
	var sealedSecret []byte
	var secretAt *time.Time
	if integrationType == Webhook {
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
		insert into work_integrations
		    (id, owner_id, type, name, host, address, signing_secret, signing_secret_set_at,
		     state, guild_id, channel_id, verified_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, id, owner, integrationType, name, prepared.host, prepared.sealed, sealedSecret, secretAt,
		prepared.state, prepared.guildID, prepared.channelID, prepared.verifiedAt)
	if err != nil {
		return Added{}, fmt.Errorf("create work update integration: %w", err)
	}
	found, err := s.Get(ctx, owner, id)
	return Added{Integration: found, Secret: secret}, err
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Integration, error) {
	rows, err := s.pool.Query(ctx, selectIntegrations+` where owner_id = $1 order by name, id`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := make([]Integration, 0)
	for rows.Next() {
		one, err := readIntegration(rows)
		if err != nil {
			return nil, err
		}
		found = append(found, one)
	}
	return found, rows.Err()
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (Integration, error) {
	return readIntegration(s.pool.QueryRow(ctx, selectIntegrations+` where owner_id = $1 and id = $2`, owner, id))
}

const selectIntegrations = `
	select id, type, name, host, state, guild_id, channel_id, signing_secret_set_at,
	       previous_secret_until, verified_at, disabled_at, created_at, version
	  from work_integrations
`

func readIntegration(row pgx.Row) (Integration, error) {
	var d Integration
	err := row.Scan(&d.ID, &d.Type, &d.Name, &d.Host, &d.State, &d.GuildID, &d.ChannelID,
		&d.SecretSetAt, &d.PreviousSecretUntil, &d.VerifiedAt, &d.DisabledAt, &d.CreatedAt, &d.version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Integration{}, ErrNotFound
	}
	d.Address = "https://" + d.Host + "/…"
	return d, err
}
