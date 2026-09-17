package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	errIngestLeaseLost = errors.New("ingest lease lost")
	errWrongKind       = errors.New("revision resolves to a different kind")
)

func (s *Service) RunIngestWorkers(ctx context.Context, count int, report func(error)) {
	var workers sync.WaitGroup
	workers.Add(count)
	for range count {
		go func() {
			defer workers.Done()
			s.runIngestWorker(ctx, report)
		}()
	}
	workers.Wait()
}

func (s *Service) runIngestWorker(ctx context.Context, report func(error)) {
	for {
		processed, err := s.ProcessNextIngest(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			if report != nil {
				report(err)
			}
		}
		if err == nil && processed {
			continue
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

type ingestJob struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	BlobID     uuid.UUID
	LeaseToken uuid.UUID
	Filename   string
	Name       *string
	Blurb      *string
	Tags       []string
	IsNSFW     *bool
	Discovery  Discovery
	ByteSize   int64
	Attempts   int
	Target     *revisionTarget
}

type revisionTarget struct {
	Version int64
	AssetID uuid.UUID
	Kind    string
}

type preparedIngest struct {
	Kind          string
	Format        string
	Name          string
	Blurb         string
	Tags          []string
	IsNSFW        bool
	Discovery     Discovery
	Blocks        []block.Block
	SuppliedRoles []block.Role
	Header        format.Header
	Remainder     []format.Remainder
	Protected     format.ProtectedImport
	Media         []preparedMedia
	Vault         []VaultPicture
	CreatedAt     *time.Time
	MediaType     string
}

type preparedImport struct {
	Parsed    format.Parsed
	Blocks    []block.Block
	Elements  []block.Element
	Media     []preparedMedia
	Vault     []VaultPicture
	MediaType string
}

func (s *Service) readImport(
	ctx context.Context,
	inspected probe.Inspection,
	expectedKind string,
) (preparedImport, error) {
	resolution, claimed, err := s.reg.Resolve(inspected)
	if err != nil {
		return preparedImport{}, err
	}
	if !claimed {
		return preparedImport{}, format.ErrUnsupportedFormat
	}

	declaration := resolution.Module.Declaration()
	payload, ok := resolution.Claim.Payload(inspected)
	if !ok {
		return preparedImport{}, format.ErrInvalidClaim
	}
	payloadBytes := payload.ByteSize
	if payloadBytes == 0 {
		if encoded, marshalErr := json.Marshal(payload.Root); marshalErr == nil {
			payloadBytes = int64(len(encoded))
		}
	}
	if declaration.Limits.PayloadBytes > 0 && payloadBytes > int64(declaration.Limits.PayloadBytes) {
		return preparedImport{}, format.LimitExceeded(fmt.Errorf(
			"%s reads payloads up to %d bytes; this payload has %d bytes%s",
			resolution.Module.ID(), declaration.Limits.PayloadBytes, payloadBytes,
			heaviestNamespace(payload, declaration),
		))
	}

	if files := len(inspected.ZIPEntries); declaration.Limits.ArchiveFiles > 0 && files > declaration.Limits.ArchiveFiles {
		return preparedImport{}, format.LimitExceeded(fmt.Errorf(
			"the archive holds %d files, and one may hold %d", files, declaration.Limits.ArchiveFiles,
		))
	}

	parsed, err := resolution.Module.Parse(ctx, inspected, resolution.Claim)
	if err != nil {
		if _, classified := format.FailureOf(err); classified {
			return preparedImport{}, err
		}
		return preparedImport{}, format.MalformedInput(err)
	}
	if parsed.Kind != declaration.Kind {
		return preparedImport{}, format.InternalFailure(fmt.Errorf(
			"module %q parsed kind %q instead of declared kind %q",
			resolution.Module.ID(), parsed.Kind, declaration.Kind,
		))
	}
	if parsed.Format != declaration.ID {
		return preparedImport{}, format.InternalFailure(fmt.Errorf(
			"module %q parsed format %q instead of its declared identity",
			resolution.Module.ID(), parsed.Format,
		))
	}
	if expectedKind != "" && parsed.Kind != expectedKind {
		return preparedImport{}, errWrongKind
	}
	if _, ok := block.Catalog(parsed.Kind); !ok {
		return preparedImport{}, ErrKindNotBuildable
	}

	preparedMedia, err := s.prepareExtractedMedia(ctx, inspected, parsed.Media)
	if err != nil {
		return preparedImport{}, err
	}
	elements := append(slices.Clone(parsed.Elements), elementsForExtractedMedia(preparedMedia)...)
	if err := block.ValidateContentLimits(elements); err != nil {
		return preparedImport{}, format.LimitExceeded(err)
	}
	blocks, err := block.Place(parsed.Kind, elements)
	if err != nil {
		return preparedImport{}, fmt.Errorf("place imported content: %w", err)
	}
	mediaType := inspected.InlineMediaType()
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return preparedImport{
		Parsed: parsed, Blocks: blocks, Elements: elements, Media: preparedMedia, MediaType: mediaType,
	}, nil
}

func heaviestNamespace(payload probe.Payload, declaration format.Declaration) string {
	container := payload.Root
	for _, part := range declaration.Preservation.Container {
		raw, present := container[part]
		if !present {
			return ""
		}
		var next map[string]json.RawMessage
		if json.Unmarshal(raw, &next) != nil {
			return ""
		}
		container = next
	}
	heaviest, size := "", 0
	for _, namespace := range slices.Sorted(maps.Keys(container)) {
		if held := len(container[namespace]); held > size {
			heaviest, size = namespace, held
		}
	}
	if heaviest == "" {
		return ""
	}
	return fmt.Sprintf(". The largest part of it is the %s data, at %d bytes", heaviest, size)
}

// refusal words why a file was refused as the sentence its creator reads.
func refusal(err error) string {
	message := err.Error()
	if _, cause, ok := format.Explain(err); ok {
		message = cause
	}
	first, size := utf8.DecodeRuneInString(message)
	message = string(unicode.ToUpper(first)) + message[size:]
	if !strings.HasSuffix(message, ".") {
		message += "."
	}
	return message
}

func (s *Service) ProcessNextIngest(ctx context.Context) (bool, error) {
	job, ok, err := s.leaseNextIngest(ctx)
	if err != nil || !ok {
		return ok, err
	}

	inspected, err := probe.InspectWithLimits(
		ctx, s.store, job.BlobID, job.ByteSize, job.Filename, s.ingest.ProbeLimits,
	)
	if err != nil {
		if violation := (probe.SafetyViolation{}); errors.As(err, &violation) {
			return true, s.finishIngestFailure(ctx, job, format.FailureSafetyViolation,
				"The file breaks an archive safety rule: "+violation.Rule+".")
		}
		if errors.Is(err, probe.ErrSafetyViolation) {
			return true, s.finishIngestFailure(ctx, job, format.FailureSafetyViolation)
		}
		if errors.Is(err, probe.ErrMalformedInput) {
			return true, s.finishIngestFailure(ctx, job, format.FailureMalformedInput)
		}
		return true, s.finishIngestFailure(ctx, job, format.FailureInternal)
	}
	expectedKind := ""
	if job.Target != nil {
		expectedKind = job.Target.Kind
	}
	read, err := s.readImport(ctx, inspected, expectedKind)
	if err != nil {
		if err == format.ErrUnsupportedFormat {
			return true, s.finishIngestFailure(ctx, job, format.FailureUnsupportedFormat)
		}
		reason := mediaIngestFailure(err)
		if errors.Is(err, errWrongKind) {
			reason = format.FailureWrongKind
		}
		if errors.Is(err, format.ErrUnsupportedFormat) || errors.Is(err, ErrKindNotBuildable) {
			reason = format.FailureUnsupportedFormat
		}
		if classified, ok := format.FailureOf(err); ok {
			reason = classified
		}
		return true, s.finishIngestFailure(ctx, job, reason, refusal(err))
	}
	if job.Target == nil {
		if read, err = s.seedFromReadme(ctx, inspected, read); err != nil {
			return true, s.finishIngestFailure(ctx, job, format.FailureInternal)
		}
	}

	prepared, err := prepareIngest(job, read.Parsed)
	if errors.Is(err, errWrongKind) {
		return true, s.finishIngestFailure(ctx, job, format.FailureWrongKind)
	}
	if err != nil {
		return true, s.finishIngestFailure(ctx, job, format.FailureInternal)
	}
	prepared.Blocks = read.Blocks
	prepared.SuppliedRoles = suppliedRoles(read.Elements)
	prepared.Media = read.Media
	prepared.Vault = read.Vault
	prepared.MediaType = read.MediaType
	finish := s.finalizeIngest
	if job.Target != nil {
		finish = s.stageReplacement
	}
	if err := finish(ctx, job, prepared); err != nil {
		if errors.Is(err, errIngestLeaseLost) {
			return true, nil
		}
		var conflict *VersionConflict
		if errors.As(err, &conflict) || errors.Is(err, ErrVersionRequired) {
			return true, s.failIngest(ctx, job, "working_copy_conflict", "The working copy changed. Review it before accepting the upload again.")
		}
		if errors.Is(err, ErrAssetFrozen) || errors.Is(err, ErrNotFound) {
			return true, s.failIngest(ctx, job, "asset_unavailable", "This asset is no longer available for changes.")
		}
		if errors.Is(err, ErrStorageCap) {
			return true, s.finishIngestFailure(
				ctx, job, format.FailureLimitExceeded,
				"The imported file would take this account past its storage cap.",
			)
		}
		if classified, why, ok := format.Explain(err); ok {
			return true, s.finishIngestFailure(ctx, job, classified, why)
		}
		return true, s.finishIngestFailure(ctx, job, format.FailureInternal)
	}
	return true, nil
}

func suppliedRoles(elements []block.Element) []block.Role {
	roles := make([]block.Role, 0, len(elements))
	for _, element := range elements {
		if element.Role != "" {
			roles = append(roles, element.Role)
		}
	}
	return roles
}

func (s *Service) leaseNextIngest(ctx context.Context) (ingestJob, bool, error) {
	now := s.now()
	leaseToken := uuid.New()
	leaseExpires := now.Add(s.ingest.LeaseDuration)
	row := s.pool.QueryRow(ctx, `
		with candidate as (
			select id
			  from ingest_operations
			 where available_at <= $1
			   and (status = 'pending'
			        or (status = 'processing' and lease_expires_at <= $1))
			 order by available_at, created_at
			 for update skip locked
			 limit 1
		)
		update ingest_operations operation
		   set status = 'processing',
		       attempts = attempts + 1,
		       lease_token = $2,
		       lease_expires_at = $3,
		       updated_at = $1
		  from candidate
		 where operation.id = candidate.id
		returning operation.id, operation.owner_id, operation.blob_id,
		          operation.filename, operation.name, operation.blurb,
		          operation.tags, operation.is_nsfw, operation.discovery,
		          operation.attempts,
		          (select byte_size from blobs where id = operation.blob_id),
		          operation.target_asset_id,
		          (select kind from assets where id = operation.target_asset_id), coalesce(operation.candidate_version, 0)
	`, now, leaseToken, leaseExpires)

	var job ingestJob
	var name, blurb, targetKind pgtype.Text
	var isNSFW pgtype.Bool
	var targetAssetID pgtype.UUID
	var candidateVersion int64
	err := row.Scan(
		&job.ID, &job.OwnerID, &job.BlobID, &job.Filename, &name, &blurb,
		&job.Tags, &isNSFW, &job.Discovery, &job.Attempts, &job.ByteSize,
		&targetAssetID, &targetKind, &candidateVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ingestJob{}, false, nil
	}
	if err != nil {
		return ingestJob{}, false, fmt.Errorf("lease ingest: %w", err)
	}
	job.LeaseToken = leaseToken
	if targetAssetID.Valid {
		if !targetKind.Valid {
			return ingestJob{}, false, fmt.Errorf("ingest %s targets a missing asset", job.ID)
		}
		job.Target = &revisionTarget{
			AssetID: uuidFromPgtype(targetAssetID), Kind: targetKind.String, Version: candidateVersion,
		}
	}
	job.Name = textToPointer(name)
	job.Blurb = textToPointer(blurb)
	if isNSFW.Valid {
		job.IsNSFW = &isNSFW.Bool
	}
	return job, true, nil
}

func prepareIngest(job ingestJob, parsed format.Parsed) (preparedIngest, error) {
	kind := parsed.Kind
	if kind == "" {
		return preparedIngest{}, errors.New("claimed format did not declare a kind")
	}
	if job.Target != nil && kind != job.Target.Kind {
		return preparedIngest{}, errWrongKind
	}

	name := parsed.Header.Name
	if job.Name != nil {
		name = *job.Name
	}
	blurb := parsed.Header.Blurb
	if job.Blurb != nil {
		blurb = *job.Blurb
	}
	tags := parsed.Tags
	if job.Tags != nil {
		tags = job.Tags
	}
	if tags == nil {
		tags = []string{}
	}
	isNSFW := false
	if parsed.IsNSFW != nil {
		isNSFW = *parsed.IsNSFW
	}
	if job.IsNSFW != nil {
		isNSFW = *job.IsNSFW
	}
	return preparedIngest{
		Kind: kind, Format: parsed.Format,
		Name: name, Blurb: blurb, Tags: tags, IsNSFW: isNSFW,
		Discovery: job.Discovery,
		Header:    parsed.Header,
		Remainder: parsed.Remainder,
		Protected: parsed.Protected,
		CreatedAt: parsed.CreatedAt,
	}, nil
}

func (s *Service) finishIngestFailure(
	ctx context.Context,
	job ingestJob,
	reason format.FailureReason,
	detail ...string,
) error {
	if reason == format.FailureInternal && job.Attempts < s.ingest.MaxAttempts {
		return s.retryIngest(ctx, job)
	}
	message := ""
	if len(detail) > 0 {
		message = detail[0]
	}
	return s.failIngest(ctx, job, string(reason), message)
}

func (s *Service) retryIngest(ctx context.Context, job ingestJob) error {
	delay := s.ingest.RetryBase
	for attempt := 1; attempt < job.Attempts; attempt++ {
		delay *= 2
	}
	now := s.now()
	result, err := s.pool.Exec(ctx, `
		update ingest_operations
		   set status = 'pending', available_at = $3, lease_token = null,
		       lease_expires_at = null, updated_at = $4
		 where id = $1 and lease_token = $2 and status = 'processing'
	`, job.ID, job.LeaseToken, now.Add(delay), now)
	if err != nil {
		return fmt.Errorf("retry ingest: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errIngestLeaseLost
	}
	return nil
}

func (s *Service) failIngest(ctx context.Context, job ingestJob, reason, message string) error {
	now := s.now()
	result, err := s.pool.Exec(ctx, `
		update ingest_operations
		   set status = 'failed', failure_reason = $3, failure_message = nullif($4, ''),
		       blob_id = null, lease_token = null, lease_expires_at = null, updated_at = $5
		 where id = $1 and lease_token = $2 and status = 'processing'
	`, job.ID, job.LeaseToken, reason, message, now)
	if err != nil {
		return fmt.Errorf("fail ingest: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errIngestLeaseLost
	}
	return nil
}

func (s *Service) finalizeIngest(ctx context.Context, job ingestJob, prepared preparedIngest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin ingest finalization: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockBlobDigest(ctx, tx, job.BlobID); errors.Is(err, pgx.ErrNoRows) {
		return errIngestLeaseLost
	} else if err != nil {
		return fmt.Errorf("lock ingest digest: %w", err)
	}

	var status IngestStatus
	var lease pgtype.UUID
	if err := tx.QueryRow(ctx, `
		select status, lease_token from ingest_operations where id = $1 for update
	`, job.ID).Scan(&status, &lease); err != nil {
		return fmt.Errorf("lock ingest finalization: %w", err)
	}
	if status == IngestSuccess {
		return nil
	}
	if status != IngestProcessing || !lease.Valid || uuidFromPgtype(lease) != job.LeaseToken {
		return errIngestLeaseLost
	}
	candidates := make([]uuid.UUID, 1, len(prepared.Media)+1)
	candidates[0] = job.BlobID
	for _, media := range prepared.Media {
		candidates = append(candidates, media.BlobID)
	}
	if err := s.ensureAccountStorage(ctx, tx, job.OwnerID, candidates); err != nil {
		return err
	}

	assetID, err := s.writeIngestResult(ctx, tx, job, prepared)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		update ingest_operations
		   set status = 'success', asset_id = $3, blob_id = null,
		       lease_token = null, lease_expires_at = null, updated_at = $4
		 where id = $1 and lease_token = $2 and status = 'processing'
	`, job.ID, job.LeaseToken, assetID, s.now())
	if err != nil {
		return fmt.Errorf("finish ingest operation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errIngestLeaseLost
	}
	if job.Target != nil {
		candidate := &Candidate{Version: job.Target.Version}
		if err := candidate.Commit(ctx, tx, job.Target.AssetID); err != nil {
			return err
		}
	} else if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit ingest finalization: %w", err)
	}
	return nil
}

func importProtectedPrompts(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	blocks []block.Block,
	carried map[uuid.UUID]string,
	imported format.ProtectedImport,
) error {
	if len(imported.Prompts) == 0 {
		return nil
	}
	if err := protected.ImportPromptFragments(
		ctx, tx, assetID, blocks, carried, imported.Prompts, imported.Apps,
	); err != nil {
		return fmt.Errorf("import protected prompts: %w", err)
	}
	return nil
}

func (s *Service) writeIngestResult(
	ctx context.Context,
	tx pgx.Tx,
	job ingestJob,
	prepared preparedIngest,
) (uuid.UUID, error) {
	return s.writeIngestResultWithDecisions(ctx, tx, job, prepared, nil, false)
}

func (s *Service) writeIngestResultWithDecisions(
	ctx context.Context,
	tx pgx.Tx,
	job ingestJob,
	prepared preparedIngest,
	decisions map[string]string,
	exposeProtected bool,
) (uuid.UUID, error) {
	blocks := prepared.Blocks
	if job.Target != nil {
		candidate := &Candidate{Version: job.Target.Version}
		if _, err := candidate.Lock(ctx, tx, job.OwnerID, job.Target.AssetID); err != nil {
			return uuid.Nil, err
		}
		existing, err := block.Read(ctx, tx, job.Target.AssetID)
		if err != nil {
			return uuid.Nil, err
		}
		carried, err := carriedPromptText(ctx, tx, job.Target.AssetID, existing, prepared.Remainder)
		if err != nil {
			return uuid.Nil, err
		}
		blocks = mergeReplacementBlocks(existing, blocks, prepared.SuppliedRoles, decisions)
		if !exposeProtected {
			identities, err := stableItemNames(ctx, tx, job.Target.AssetID, prepared.Remainder)
			if err != nil {
				return uuid.Nil, err
			}
			exposed, err := protected.UnsealedReplacement(ctx, tx, job.Target.AssetID, blocks, identities, prepared.Protected.Prompts)
			if err != nil {
				return uuid.Nil, err
			}
			if len(exposed) > 0 {
				return uuid.Nil, ExposureRefusal{Prompts: exposed}
			}
		}
		fingerprint, err := s.contentFingerprint(ctx, tx, job.Target.AssetID)
		if err != nil {
			return uuid.Nil, err
		}
		if err := appendRevision(ctx, tx, job, prepared); err != nil {
			return uuid.Nil, err
		}
		if _, err := tx.Exec(ctx, `delete from asset_blocks where asset_id = $1`, job.Target.AssetID); err != nil {
			return uuid.Nil, fmt.Errorf("replace imported blocks: %w", err)
		}
		if err := insertBlocks(ctx, tx, job.Target.AssetID, blocks); err != nil {
			return uuid.Nil, err
		}
		remainder, err := retainUnrepresentableRemainder(ctx, tx, job.Target.AssetID, existing, prepared.Remainder, decisions)
		if err != nil {
			return uuid.Nil, err
		}
		if err := replacePreservedData(ctx, tx, job.Target.AssetID, remainder); err != nil {
			return uuid.Nil, err
		}
		if len(prepared.Protected.Prompts) > 0 {
			if err := importProtectedPrompts(
				ctx, tx, job.Target.AssetID, blocks, carried, prepared.Protected,
			); err != nil {
				return uuid.Nil, err
			}
		} else if err := protected.SyncPromptFragments(
			ctx, tx, job.Target.AssetID, blocks, nil,
		); err != nil {
			return uuid.Nil, fmt.Errorf("reconcile protected prompts: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			update assets
			   set origin_format = $2, asset_version = $3, credited_author = $4,
			       nickname = $5, updated_at = now()
			 where id = $1
		`, job.Target.AssetID, prepared.Format, prepared.Header.AssetVersion,
			prepared.Header.CreditedAuthor, prepared.Header.Nickname); err != nil {
			return uuid.Nil, fmt.Errorf("move asset origin: %w", err)
		}
		if err := s.writeProjections(ctx, tx, job.Target.AssetID); err != nil {
			return uuid.Nil, err
		}
		if err := s.moveContentGeneration(ctx, tx, job.Target.AssetID, fingerprint); err != nil {
			return uuid.Nil, err
		}
		return job.Target.AssetID, nil
	}
	assetID := uuid.New()
	isNSFW := prepared.IsNSFW
	a := Asset{
		ID: assetID, Kind: prepared.Kind, Format: prepared.Format,
		OriginFormat:   &prepared.Format,
		AssetVersion:   prepared.Header.AssetVersion,
		CreditedAuthor: prepared.Header.CreditedAuthor, Nickname: prepared.Header.Nickname,
		Name: prepared.Name, Blurb: prepared.Blurb, Tags: prepared.Tags,
		IsNSFW: &isNSFW, Discovery: prepared.Discovery,
		Lifecycle: LifecycleDraft,
	}
	if _, err := insertAsset(ctx, tx, a, job.OwnerID, prepared.CreatedAt); err != nil {
		return uuid.Nil, err
	}
	if err := insertBlocks(ctx, tx, assetID, blocks); err != nil {
		return uuid.Nil, err
	}
	if err := replacePreservedData(ctx, tx, assetID, prepared.Remainder); err != nil {
		return uuid.Nil, err
	}
	if err := importProtectedPrompts(
		ctx, tx, assetID, blocks, nil, prepared.Protected,
	); err != nil {
		return uuid.Nil, err
	}
	if err := writeRevision(ctx, tx, assetID, 1, job, prepared); err != nil {
		return uuid.Nil, err
	}
	if err := insertVaultPictures(ctx, tx, assetID, prepared.Vault); err != nil {
		return uuid.Nil, err
	}
	return assetID, s.writeProjections(ctx, tx, assetID)
}

func replacePreservedData(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	remainder []format.Remainder,
) error {
	if _, err := tx.Exec(ctx, `delete from asset_preserved_data where asset_id = $1`, assetID); err != nil {
		return fmt.Errorf("replace preserved data: %w", err)
	}
	for _, item := range remainder {
		if item.Namespace == "" || len(item.Payload) == 0 {
			return errors.New("preserved data needs a namespace and payload")
		}
		owner := assetID
		switch item.Owner {
		case format.OwnerAsset:
		case format.OwnerElement, format.OwnerItem:
			if item.OwnerID == uuid.Nil {
				return fmt.Errorf("preserved %s names no %s to belong to", item.Namespace, item.Owner)
			}
			owner = item.OwnerID
		default:
			return fmt.Errorf("preserved %s belongs to %q", item.Namespace, item.Owner)
		}
		if _, err := tx.Exec(ctx, `
			insert into asset_preserved_data
			  (id, asset_id, owner_kind, owner_id, namespace, payload)
			values ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), assetID, string(item.Owner), owner, item.Namespace, item.Payload); err != nil {
			return fmt.Errorf("preserve %s: %w", item.Namespace, err)
		}
	}
	return nil
}

func appendRevision(
	ctx context.Context,
	tx pgx.Tx,
	job ingestJob,
	prepared preparedIngest,
) error {
	var next int
	if err := tx.QueryRow(ctx, `
		select coalesce(max(revision), 0) + 1 from asset_revisions where asset_id = $1
	`, job.Target.AssetID).Scan(&next); err != nil {
		return fmt.Errorf("number the new revision: %w", err)
	}
	return writeRevision(ctx, tx, job.Target.AssetID, next, job, prepared)
}

func writeRevision(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	number int,
	job ingestJob,
	prepared preparedIngest,
) error {
	revisionID := uuid.New()
	if err := insertRevision(ctx, tx, revisionID, assetID, revisionRow{
		Revision: number, BlobID: job.BlobID, MediaType: prepared.MediaType,
		Format: prepared.Format, Identifier: prepared.Header.Identifier,
	}); err != nil {
		return err
	}
	if err := supersedeExtractedMedia(ctx, tx, assetID); err != nil {
		return err
	}
	if err := insertAssetMedia(ctx, tx, assetID, prepared.Media); err != nil {
		return err
	}
	if err := setCurrentRevision(ctx, tx, assetID, revisionID); err != nil {
		return err
	}
	if coverID := avatarMedia(prepared.Media); coverID != nil {
		return setCoverMedia(ctx, tx, assetID, coverID)
	}
	return clearSupersededCover(ctx, tx, assetID)
}
