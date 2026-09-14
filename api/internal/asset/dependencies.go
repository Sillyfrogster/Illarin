package asset

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
)

type DependencyAsset struct {
	ID      uuid.UUID
	Name    string
	Creator string
}

type ExtensionDependency struct {
	Name   string
	Assets []DependencyAsset
}

// dependencySubject is the page whose dependencies are matched, and the format they are matched within.
type dependencySubject struct {
	assetID    uuid.UUID
	kind       string
	format     string
	visibility ContentVisibility
}

// extensionDependencies pairs each dependency the archive names with the listed assets whose identifier it refers to.
func extensionDependencies(ctx context.Context, q db.DBTX, subject dependencySubject, blocks []block.Block) ([]ExtensionDependency, error) {
	items := dependencyItems(blocks)
	dependencies := make([]ExtensionDependency, len(items))
	identifiers := make([]string, 0, len(items))
	for i, item := range items {
		dependencies[i] = ExtensionDependency{Name: item.Text, Assets: []DependencyAsset{}}
		if item.Name != "" {
			identifiers = append(identifiers, item.Name)
		}
	}
	if len(identifiers) == 0 || subject.format == "" {
		return dependencies, nil
	}
	rows, err := q.Query(ctx, `
		select revision.identifier, listed.id, listed.name, coalesce(owner.username, 'unknown')
		  from asset_public.assets listed
		  join public.asset_revisions revision on revision.id = listed.current_revision_id
		  left join public.users owner on owner.id = listed.owner_id
		 where revision.identifier = any($1::text[])
		   and revision.format = $2 and listed.kind = $3 and listed.id <> $4
		   and listed.lifecycle = 'published' and listed.discovery = 'listed'
		   and listed.withheld_at is null and listed.deleted_at is null
		   and ($5 <> 'hidden' or not listed.is_nsfw)
		 order by listed.created_at, listed.id
	`, identifiers, subject.format, subject.kind, subject.assetID, string(subject.visibility))
	if err != nil {
		return nil, fmt.Errorf("match extension dependencies: %w", err)
	}
	defer rows.Close()
	matches := make(map[string][]DependencyAsset)
	for rows.Next() {
		var identifier string
		var found DependencyAsset
		if err := rows.Scan(&identifier, &found.ID, &found.Name, &found.Creator); err != nil {
			return nil, fmt.Errorf("read a matched extension dependency: %w", err)
		}
		matches[identifier] = append(matches[identifier], found)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("match extension dependencies: %w", err)
	}
	for i, item := range items {
		if found, ok := matches[item.Name]; ok && item.Name != "" {
			dependencies[i].Assets = found
		}
	}
	return dependencies, nil
}

func dependencyItems(blocks []block.Block) []block.TextItem {
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			if set, ok := element.Content.(block.TextSet); ok && element.Role == block.RoleExtensionDependencies {
				return set.Texts
			}
		}
	}
	return nil
}
