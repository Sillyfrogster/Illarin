package private

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const preservedPromptsSource = "preserved_prompts"

var ErrNotFound = errors.New("no preserved prompts")

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type PreservedPromptsFile struct {
	Body      []byte
	MediaType string
	Filename  string
	Blocks    int
}

type preservedPromptsExport struct {
	WorkID   uuid.UUID         `json:"work_id"`
	WorkName string            `json:"work_name"`
	Source   string            `json:"source"`
	Blocks   []json.RawMessage `json:"blocks"`
}

func (s *Service) OpenPreservedPrompts(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) (PreservedPromptsFile, error) {
	rows, err := s.pool.Query(ctx, `
		select owned.name, record.payload
		  from migration_preserved_records record
		  join works owned on owned.id = record.work_id
		 where record.work_id = $1
		   and record.source_table = $2
		   and owned.owner_id = $3
		   and owned.deleted_at is null
		 order by record.payload ->> 'version', record.payload ->> 'block_key'
	`, workID, preservedPromptsSource, ownerID)
	if err != nil {
		return PreservedPromptsFile{}, fmt.Errorf("read preserved prompts: %w", err)
	}
	defer rows.Close()

	name := ""
	blocks := make([]json.RawMessage, 0)
	for rows.Next() {
		var payload json.RawMessage
		if err := rows.Scan(&name, &payload); err != nil {
			return PreservedPromptsFile{}, fmt.Errorf("read a preserved prompt: %w", err)
		}
		blocks = append(blocks, payload)
	}
	if err := rows.Err(); err != nil {
		return PreservedPromptsFile{}, fmt.Errorf("read preserved prompts: %w", err)
	}
	if len(blocks) == 0 {
		return PreservedPromptsFile{}, ErrNotFound
	}

	body, err := json.MarshalIndent(preservedPromptsExport{
		WorkID: workID, WorkName: name, Source: preservedPromptsSource, Blocks: blocks,
	}, "", "  ")
	if err != nil {
		return PreservedPromptsFile{}, fmt.Errorf("write preserved prompts: %w", err)
	}
	return PreservedPromptsFile{
		Body:      body,
		MediaType: "application/json",
		Filename:  format.Filename(name, "", "preserved-prompts", ".json"),
		Blocks:    len(blocks),
	}, nil
}

// PreservedPromptCount counts the preserved prompts an owner can still export from a work
func PreservedPromptCount(
	ctx context.Context,
	q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	ownerID uuid.UUID,
	workID uuid.UUID,
) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		select count(*)
		  from migration_preserved_records record
		  join works owned on owned.id = record.work_id
		 where record.work_id = $1
		   and record.source_table = $2
		   and owned.owner_id = $3
		   and owned.deleted_at is null
	`, workID, preservedPromptsSource, ownerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count preserved prompts: %w", err)
	}
	return count, nil
}
