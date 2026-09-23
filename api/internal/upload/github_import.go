package upload

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) importGitHubRelease(ctx context.Context, tx pgx.Tx, versions *version.Service, ownerID, workID uuid.UUID, filename, tag string, archive []byte) (version.Version, error) {
	stored, err := s.store.Put(ctx, bytes.NewReader(archive))
	if err != nil {
		return version.Version{}, err
	}
	inspected, err := format.InspectWithLimits(ctx, s.store, stored.ID, stored.ByteSize, filename, s.settings.ProbeLimits)
	if err != nil {
		return version.Version{}, err
	}
	read, err := s.readImport(ctx, inspected, "extension")
	if err != nil {
		return version.Version{}, err
	}
	job := uploadJob{OwnerID: ownerID, BlobID: stored.ID, Filename: filename,
		Target: &originalFileTarget{WorkID: workID, Type: "extension"}}
	prepared, err := prepareUpload(job, read.Parsed)
	if err != nil {
		return version.Version{}, err
	}
	prepared.Blocks = read.Blocks
	prepared.SuppliedRoles = suppliedRoles(read.Elements)
	prepared.Media = read.Media
	prepared.MediaType = read.MediaType

	if err := s.works.EnsureAccountStorage(ctx, tx, ownerID, []uuid.UUID{stored.ID}); err != nil {
		return version.Version{}, err
	}
	if err := tx.QueryRow(ctx, `select drafted_changes_version from works where id = $1`, workID).Scan(&job.Target.Version); err != nil {
		return version.Version{}, err
	}
	if _, err := s.writeUploadResultWithDecisions(ctx, tx, job, prepared, nil, false); err != nil {
		return version.Version{}, err
	}
	published, _, err := versions.PublishLocked(ctx, tx, version.PublishRequest{
		OwnerID: ownerID, WorkID: workID, Summary: fmt.Sprintf("Imported GitHub release %s", tag),
		VersionLabel: prepared.Header.WorkVersion, Announcement: version.Announcement{Notify: true},
	})
	if err != nil {
		return version.Version{}, err
	}
	if _, err := tx.Exec(ctx, `update works set drafted_changes_version = drafted_changes_version + 1 where id = $1`, workID); err != nil {
		return version.Version{}, err
	}
	return published, nil
}
