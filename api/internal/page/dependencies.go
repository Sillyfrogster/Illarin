package page

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type DependencyMatch struct {
	ID      uuid.UUID
	Name    string
	Creator string
}

type Dependency struct {
	Name  string
	Works []DependencyMatch
}

// dependencySubject is the page whose dependencies are matched, and the format they are matched within
type dependencySubject struct {
	workID     uuid.UUID
	workType   string
	format     string
	preference work.NSFWPreference
}

// extensionDependencies pairs each dependency the archive names with the listed works whose identifier it refers to
func extensionDependencies(ctx context.Context, q db.DBTX, subject dependencySubject, blocks []block.Block) ([]Dependency, error) {
	items := dependencyItems(blocks)
	dependencies := make([]Dependency, len(items))
	identifiers := make([]string, 0, len(items))
	for i, item := range items {
		dependencies[i] = Dependency{Name: item.Text, Works: []DependencyMatch{}}
		if item.Name != "" {
			identifiers = append(identifiers, item.Name)
		}
	}
	if len(identifiers) == 0 || subject.format == "" {
		return dependencies, nil
	}
	rows, err := q.Query(ctx, `
		select original.identifier, listed.id, listed.name, coalesce(owner.username, 'unknown')
		  from work_public.works listed
		  join public.work_original_files original on original.id = listed.original_file_id
		  left join public.users owner on owner.id = listed.owner_id
		 where original.identifier = any($1::text[])
		   and original.format = $2 and listed.type = $3 and listed.id <> $4
		   and listed.lifecycle = 'published' and listed.visibility = 'listed'
		   and listed.taken_down_at is null and listed.deleted_at is null
		   and ($5 <> 'hidden' or not listed.is_nsfw)
		 order by listed.created_at, listed.id
	`, identifiers, subject.format, subject.workType, subject.workID, string(subject.preference))
	if err != nil {
		return nil, fmt.Errorf("match extension dependencies: %w", err)
	}
	defer rows.Close()
	matches := make(map[string][]DependencyMatch)
	for rows.Next() {
		var identifier string
		var found DependencyMatch
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
			dependencies[i].Works = found
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
