package upload

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
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	errIngestLeaseLost = errors.New("ingest lease lost")
	errWrongType       = errors.New("revision resolves to a different type")
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
	Visibility work.Visibility
	ByteSize   int64
	Attempts   int
	Target     *revisionTarget
}

type revisionTarget struct {
	Version int64
	WorkID  uuid.UUID
	Type    string
}

type preparedIngest struct {
	Type          string
	Format        string
	Name          string
	Blurb         string
	Tags          []string
	IsNSFW        bool
	Visibility    work.Visibility
	Blocks        []block.Block
	SuppliedRoles []block.Role
	Header        format.Header
	Remainder     []format.Remainder
	Protected     format.ProtectedImport
	Media         []work.PreparedMedia
	Vault         []WaitingPicture
	CreatedAt     *time.Time
	MediaType     string
}

type preparedImport struct {
	Parsed    format.Parsed
	Blocks    []block.Block
	Elements  []block.Element
	Media     []work.PreparedMedia
	Vault     []WaitingPicture
	MediaType string
}

func (s *Service) readImport(
	ctx context.Context,
	inspected format.Inspection,
	expectedType string,
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
	if parsed.Type != declaration.Type {
		return preparedImport{}, format.InternalFailure(fmt.Errorf(
			"module %q parsed kind %q instead of declared kind %q",
			resolution.Module.ID(), parsed.Type, declaration.Type,
		))
	}
	if parsed.Format != declaration.ID {
		return preparedImport{}, format.InternalFailure(fmt.Errorf(
			"module %q parsed format %q instead of its declared identity",
			resolution.Module.ID(), parsed.Format,
		))
	}
	if expectedType != "" && parsed.Type != expectedType {
		return preparedImport{}, errWrongType
	}
	if _, ok := block.Definitions(parsed.Type); !ok {
		return preparedImport{}, ErrTypeNotBuildable
	}

	extracted, err := s.works.PrepareExtractedMedia(ctx, inspected, parsed.Media)
	if err != nil {
		return preparedImport{}, err
	}
	elements := append(slices.Clone(parsed.Elements), work.ElementsForExtractedMedia(extracted)...)
	if err := block.ValidateContentLimits(elements); err != nil {
		return preparedImport{}, format.LimitExceeded(err)
	}
	blocks, err := block.Place(parsed.Type, elements)
	if err != nil {
		return preparedImport{}, fmt.Errorf("place imported content: %w", err)
	}
	mediaType := inspected.InlineMediaType()
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return preparedImport{
		Parsed: parsed, Blocks: blocks, Elements: elements, Media: extracted, MediaType: mediaType,
	}, nil
}

func heaviestNamespace(payload format.Payload, declaration format.Declaration) string {
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

// refusal words why a file was refused as a sentence for its creator
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

	inspected, err := format.InspectWithLimits(
		ctx, s.store, job.BlobID, job.ByteSize, job.Filename, s.settings.ProbeLimits,
	)
	if err != nil {
		if violation := (format.ArchiveViolation{}); errors.As(err, &violation) {
			return true, s.finishIngestFailure(ctx, job, format.FailureSafetyViolation,
				"The file breaks an archive safety rule: "+violation.Rule+".")
		}
		if errors.Is(err, format.ErrSafetyViolation) {
			return true, s.finishIngestFailure(ctx, job, format.FailureSafetyViolation)
		}
		if errors.Is(err, format.ErrMalformedInput) {
			return true, s.finishIngestFailure(ctx, job, format.FailureMalformedInput)
		}
		return true, s.finishIngestFailure(ctx, job, format.FailureInternal)
	}
	expectedType := ""
	if job.Target != nil {
		expectedType = job.Target.Type
	}
	read, err := s.readImport(ctx, inspected, expectedType)
	if err != nil {
		if err == format.ErrUnsupportedFormat {
			return true, s.finishIngestFailure(ctx, job, format.FailureUnsupportedFormat)
		}
		reason := work.MediaIngestFailure(err)
		if errors.Is(err, errWrongType) {
			reason = format.FailureWrongType
		}
		if errors.Is(err, format.ErrUnsupportedFormat) || errors.Is(err, ErrTypeNotBuildable) {
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
	if errors.Is(err, errWrongType) {
		return true, s.finishIngestFailure(ctx, job, format.FailureWrongType)
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
		var conflict *work.VersionConflict
		if errors.As(err, &conflict) || errors.Is(err, work.ErrVersionRequired) {
			return true, s.failIngest(ctx, job, "working_copy_conflict", "The working copy changed. Review it before accepting the upload again.")
		}
		if errors.Is(err, work.ErrWorkFrozen) || errors.Is(err, work.ErrNotFound) {
			return true, s.failIngest(ctx, job, "asset_unavailable", "This asset is no longer available for changes.")
		}
		if errors.Is(err, work.ErrStorageCap) {
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
	leaseExpires := now.Add(s.settings.LeaseDuration)
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
		          operation.tags, operation.is_nsfw, operation.visibility,
		          operation.attempts,
		          (select byte_size from blobs where id = operation.blob_id),
		          operation.target_work_id,
		          (select type from works where id = operation.target_work_id), coalesce(operation.candidate_version, 0)
	`, now, leaseToken, leaseExpires)

	var job ingestJob
	var name, blurb, targetType pgtype.Text
	var isNSFW pgtype.Bool
	var targetWorkID pgtype.UUID
	var candidateVersion int64
	err := row.Scan(
		&job.ID, &job.OwnerID, &job.BlobID, &job.Filename, &name, &blurb,
		&job.Tags, &isNSFW, &job.Visibility, &job.Attempts, &job.ByteSize,
		&targetWorkID, &targetType, &candidateVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ingestJob{}, false, nil
	}
	if err != nil {
		return ingestJob{}, false, fmt.Errorf("lease ingest: %w", err)
	}
	job.LeaseToken = leaseToken
	if targetWorkID.Valid {
		if !targetType.Valid {
			return ingestJob{}, false, fmt.Errorf("ingest %s targets a missing work", job.ID)
		}
		job.Target = &revisionTarget{
			WorkID: uuidFromPgtype(targetWorkID), Type: targetType.String, Version: candidateVersion,
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
	workType := parsed.Type
	if workType == "" {
		return preparedIngest{}, errors.New("claimed format did not declare a type")
	}
	if job.Target != nil && workType != job.Target.Type {
		return preparedIngest{}, errWrongType
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
		Type: workType, Format: parsed.Format,
		Name: name, Blurb: blurb, Tags: tags, IsNSFW: isNSFW,
		Visibility: job.Visibility,
		Header:     parsed.Header,
		Remainder:  parsed.Remainder,
		Protected:  parsed.Protected,
		CreatedAt:  parsed.CreatedAt,
	}, nil
}

func (s *Service) finishIngestFailure(
	ctx context.Context,
	job ingestJob,
	reason format.FailureReason,
	detail ...string,
) error {
	if reason == format.FailureInternal && job.Attempts < s.settings.MaxAttempts {
		return s.retryIngest(ctx, job)
	}
	message := ""
	if len(detail) > 0 {
		message = detail[0]
	}
	return s.failIngest(ctx, job, string(reason), message)
}

func (s *Service) retryIngest(ctx context.Context, job ingestJob) error {
	delay := s.settings.RetryBase
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
	if err := storage.LockBlobDigest(ctx, tx, job.BlobID); errors.Is(err, pgx.ErrNoRows) {
		return errIngestLeaseLost
	} else if err != nil {
		return fmt.Errorf("lock ingest digest: %w", err)
	}

	var status Status
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
	if err := s.works.EnsureAccountStorage(ctx, tx, job.OwnerID, candidates); err != nil {
		return err
	}

	workID, err := s.writeIngestResult(ctx, tx, job, prepared)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		update ingest_operations
		   set status = 'success', work_id = $3, blob_id = null,
		       lease_token = null, lease_expires_at = null, updated_at = $4
		 where id = $1 and lease_token = $2 and status = 'processing'
	`, job.ID, job.LeaseToken, workID, s.now())
	if err != nil {
		return fmt.Errorf("finish ingest operation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errIngestLeaseLost
	}
	if job.Target != nil {
		candidate := &work.Candidate{Version: job.Target.Version}
		if err := candidate.Commit(ctx, tx, job.Target.WorkID); err != nil {
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
	workID uuid.UUID,
	blocks []block.Block,
	carried map[uuid.UUID]string,
	imported format.ProtectedImport,
) error {
	if len(imported.Prompts) == 0 {
		return nil
	}
	if err := private.ImportPromptFragments(
		ctx, tx, workID, blocks, carried, imported.Prompts, imported.Apps,
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
		candidate := &work.Candidate{Version: job.Target.Version}
		if _, err := candidate.Lock(ctx, tx, job.OwnerID, job.Target.WorkID); err != nil {
			return uuid.Nil, err
		}
		existing, err := block.Read(ctx, tx, job.Target.WorkID)
		if err != nil {
			return uuid.Nil, err
		}
		carried, err := carriedPromptText(ctx, tx, job.Target.WorkID, existing, prepared.Remainder)
		if err != nil {
			return uuid.Nil, err
		}
		blocks = mergeReplacementBlocks(existing, blocks, prepared.SuppliedRoles, decisions)
		if !exposeProtected {
			identities, err := stableItemNames(ctx, tx, job.Target.WorkID, prepared.Remainder)
			if err != nil {
				return uuid.Nil, err
			}
			exposed, err := private.UnsealedReplacement(ctx, tx, job.Target.WorkID, blocks, identities, prepared.Protected.Prompts)
			if err != nil {
				return uuid.Nil, err
			}
			if len(exposed) > 0 {
				return uuid.Nil, private.ExposureRefusal{Prompts: exposed}
			}
		}
		change := func() error {
			return s.replaceContent(ctx, tx, job, prepared, existing, blocks, carried, decisions)
		}
		if err := s.works.ChangeContent(ctx, tx, job.Target.WorkID, change); err != nil {
			return uuid.Nil, err
		}
		return job.Target.WorkID, nil
	}
	workID := uuid.New()
	isNSFW := prepared.IsNSFW
	a := work.Work{
		ID: workID, Type: prepared.Type, Format: prepared.Format,
		OriginFormat:   &prepared.Format,
		WorkVersion:    prepared.Header.WorkVersion,
		CreditedAuthor: prepared.Header.CreditedAuthor, Nickname: prepared.Header.Nickname,
		Name: prepared.Name, Blurb: prepared.Blurb, Tags: prepared.Tags,
		IsNSFW: &isNSFW, Visibility: prepared.Visibility,
		Lifecycle: work.LifecycleDraft,
	}
	if _, err := work.InsertWork(ctx, tx, a, job.OwnerID, prepared.CreatedAt); err != nil {
		return uuid.Nil, err
	}
	if err := block.Insert(ctx, tx, workID, blocks); err != nil {
		return uuid.Nil, err
	}
	if err := replacePreservedData(ctx, tx, workID, prepared.Remainder); err != nil {
		return uuid.Nil, err
	}
	if err := importProtectedPrompts(
		ctx, tx, workID, blocks, nil, prepared.Protected,
	); err != nil {
		return uuid.Nil, err
	}
	if err := writeRevision(ctx, tx, workID, 1, job, prepared); err != nil {
		return uuid.Nil, err
	}
	if err := insertVaultPictures(ctx, tx, workID, prepared.Vault); err != nil {
		return uuid.Nil, err
	}
	return workID, s.writeSummary(ctx, tx, workID)
}

func (s *Service) replaceContent(
	ctx context.Context,
	tx pgx.Tx,
	job ingestJob,
	prepared preparedIngest,
	existing, blocks []block.Block,
	carried map[uuid.UUID]string,
	decisions map[string]string,
) error {
	if err := appendRevision(ctx, tx, job, prepared); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from work_blocks where work_id = $1`, job.Target.WorkID); err != nil {
		return fmt.Errorf("replace imported blocks: %w", err)
	}
	if err := block.Insert(ctx, tx, job.Target.WorkID, blocks); err != nil {
		return err
	}
	remainder, err := retainUnrepresentableRemainder(ctx, tx, job.Target.WorkID, existing, prepared.Remainder, decisions)
	if err != nil {
		return err
	}
	if err := replacePreservedData(ctx, tx, job.Target.WorkID, remainder); err != nil {
		return err
	}
	if len(prepared.Protected.Prompts) > 0 {
		if err := importProtectedPrompts(
			ctx, tx, job.Target.WorkID, blocks, carried, prepared.Protected,
		); err != nil {
			return err
		}
	} else if err := private.SyncPromptFragments(
		ctx, tx, job.Target.WorkID, blocks, nil,
	); err != nil {
		return fmt.Errorf("reconcile protected prompts: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		update works
		   set origin_format = $2, work_version = $3, credited_author = $4,
		       nickname = $5, updated_at = now()
		 where id = $1
	`, job.Target.WorkID, prepared.Format, prepared.Header.WorkVersion,
		prepared.Header.CreditedAuthor, prepared.Header.Nickname); err != nil {
		return fmt.Errorf("move work origin: %w", err)
	}
	if err := s.writeSummary(ctx, tx, job.Target.WorkID); err != nil {
		return err
	}
	return nil
}

func replacePreservedData(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	remainder []format.Remainder,
) error {
	if _, err := tx.Exec(ctx, `delete from work_preserved_data where work_id = $1`, workID); err != nil {
		return fmt.Errorf("replace preserved data: %w", err)
	}
	for _, item := range remainder {
		if item.Namespace == "" || len(item.Payload) == 0 {
			return errors.New("preserved data needs a namespace and payload")
		}
		owner := workID
		switch item.Owner {
		case format.OwnerWork:
		case format.OwnerElement, format.OwnerItem:
			if item.OwnerID == uuid.Nil {
				return fmt.Errorf("preserved %s names no %s to belong to", item.Namespace, item.Owner)
			}
			owner = item.OwnerID
		default:
			return fmt.Errorf("preserved %s belongs to %q", item.Namespace, item.Owner)
		}
		if _, err := tx.Exec(ctx, `
			insert into work_preserved_data
			  (id, work_id, owner_type, owner_id, namespace, payload)
			values ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), workID, string(item.Owner), owner, item.Namespace, item.Payload); err != nil {
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
		select coalesce(max(revision), 0) + 1 from work_revisions where work_id = $1
	`, job.Target.WorkID).Scan(&next); err != nil {
		return fmt.Errorf("number the new revision: %w", err)
	}
	return writeRevision(ctx, tx, job.Target.WorkID, next, job, prepared)
}

func writeRevision(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	number int,
	job ingestJob,
	prepared preparedIngest,
) error {
	_, err := work.RecordRevision(ctx, tx, work.Revision{
		WorkID: workID, Number: number, BlobID: job.BlobID, MediaType: prepared.MediaType,
		Format: prepared.Format, Identifier: prepared.Header.Identifier, Media: prepared.Media,
	})
	return err
}
