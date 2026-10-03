package upload

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// readRelease stores a release archive and reads it as an extension, for a new draft when target is nil
func (s *Service) readRelease(ctx context.Context, tx pgx.Tx, ownerID uuid.UUID, target *originalFileTarget, filename string, archive []byte) (uploadJob, preparedUpload, error) {
	stored, err := s.store.Put(ctx, bytes.NewReader(archive))
	if err != nil {
		return uploadJob{}, preparedUpload{}, err
	}
	inspected, err := format.InspectWithLimits(ctx, s.store, stored.ID, stored.ByteSize, filename, s.settings.ProbeLimits)
	if err != nil {
		return uploadJob{}, preparedUpload{}, err
	}
	read, err := s.readImport(ctx, inspected, "extension")
	if err == nil && target == nil {
		read, err = s.seedFromReadme(ctx, inspected, read)
	}
	if err != nil {
		return uploadJob{}, preparedUpload{}, err
	}
	job := uploadJob{OwnerID: ownerID, BlobID: stored.ID, Filename: filename, Visibility: work.VisibilityListed, Target: target}
	prepared, err := prepareUpload(job, read.Parsed)
	if err != nil {
		return uploadJob{}, preparedUpload{}, err
	}
	prepared.Blocks = read.Blocks
	prepared.SuppliedRoles = suppliedRoles(read.Elements)
	prepared.Media = read.Media
	prepared.Shelf = read.Shelf
	prepared.MediaType = read.MediaType

	stores := []uuid.UUID{stored.ID}
	for _, media := range read.Media {
		stores = append(stores, media.BlobID)
	}
	if err := s.works.EnsureAccountStorage(ctx, tx, ownerID, stores); err != nil {
		return uploadJob{}, preparedUpload{}, err
	}
	return job, prepared, nil
}

// startFromRelease makes a new extension draft from a release archive
func (s *Service) startFromRelease(ctx context.Context, tx pgx.Tx, ownerID uuid.UUID, filename string, archive []byte) (Started, error) {
	job, prepared, err := s.readRelease(ctx, tx, ownerID, nil, filename, archive)
	if err != nil {
		return Started{}, err
	}
	id, err := s.writeUploadResult(ctx, tx, job, prepared)
	return Started{ID: id, Name: prepared.Name}, err
}

func (s *Service) importGitHubRelease(ctx context.Context, tx pgx.Tx, versions *version.Service, ownerID, workID uuid.UUID, filename, tag string, archive []byte) (version.Version, error) {
	job, prepared, err := s.readRelease(ctx, tx, ownerID, &originalFileTarget{WorkID: workID, Type: "extension"}, filename, archive)
	if err != nil {
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
