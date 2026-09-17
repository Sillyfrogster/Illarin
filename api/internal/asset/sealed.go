package asset

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const sealedSourceTable = "preset_sealed_blocks"

type SealedContent struct {
	Body      []byte
	MediaType string
	Filename  string
	Blocks    int
}

type sealedExport struct {
	AssetID   uuid.UUID         `json:"asset_id"`
	AssetName string            `json:"asset_name"`
	Source    string            `json:"source"`
	Blocks    []json.RawMessage `json:"blocks"`
}

func (s *Service) OpenSealedContent(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
) (SealedContent, error) {
	rows, err := s.pool.Query(ctx, `
		select owned.name, record.payload
		  from migration_preserved_records record
		  join assets owned on owned.id = record.asset_id
		 where record.asset_id = $1
		   and record.source_table = $2
		   and owned.owner_id = $3
		   and owned.deleted_at is null
		 order by record.payload ->> 'version', record.payload ->> 'block_key'
	`, assetID, sealedSourceTable, ownerID)
	if err != nil {
		return SealedContent{}, fmt.Errorf("read sealed content: %w", err)
	}
	defer rows.Close()

	name := ""
	blocks := make([]json.RawMessage, 0)
	for rows.Next() {
		var payload json.RawMessage
		if err := rows.Scan(&name, &payload); err != nil {
			return SealedContent{}, fmt.Errorf("read a sealed block: %w", err)
		}
		blocks = append(blocks, payload)
	}
	if err := rows.Err(); err != nil {
		return SealedContent{}, fmt.Errorf("read sealed content: %w", err)
	}
	if len(blocks) == 0 {
		return SealedContent{}, ErrNotFound
	}

	body, err := json.MarshalIndent(sealedExport{
		AssetID: assetID, AssetName: name, Source: sealedSourceTable, Blocks: blocks,
	}, "", "  ")
	if err != nil {
		return SealedContent{}, fmt.Errorf("write sealed content: %w", err)
	}
	return SealedContent{
		Body:      body,
		MediaType: "application/json",
		Filename:  format.Filename(name, "", "sealed", ".json"),
		Blocks:    len(blocks),
	}, nil
}

// SealedBlockCount counts the preserved prompts an owner can still export from a work
func SealedBlockCount(
	ctx context.Context,
	q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	ownerID uuid.UUID,
	assetID uuid.UUID,
) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		select count(*)
		  from migration_preserved_records record
		  join assets owned on owned.id = record.asset_id
		 where record.asset_id = $1
		   and record.source_table = $2
		   and owned.owner_id = $3
		   and owned.deleted_at is null
	`, assetID, sealedSourceTable, ownerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sealed content: %w", err)
	}
	return count, nil
}

type ExposureRefusal struct {
	Prompts []string
}

func (refusal ExposureRefusal) Error() string {
	return "making a sealed prompt public needs an explicit confirmation"
}
