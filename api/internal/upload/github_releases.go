package upload

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GitHubReleases struct {
	pool     *pgxpool.Pool
	uploads  *Service
	versions *version.Service
	github   githubClient
}

type releaseSourceError string

func (e releaseSourceError) Error() string { return string(e) }

var errGitHubImportHeld = errors.New("unpublished edits exist")

func NewGitHubReleases(uploads *Service, versions *version.Service, clients ...*http.Client) *GitHubReleases {
	client := newGitHubClient()
	if len(clients) > 0 && clients[0] != nil {
		client.http = clients[0]
	}
	return &GitHubReleases{pool: uploads.pool, uploads: uploads, versions: versions, github: client}
}

type releaseSource struct {
	Repository         string          `json:"repository"`
	Attachment         *string         `json:"attachment"`
	IncludePrereleases bool            `json:"includePrereleases"`
	Verified           bool            `json:"verified"`
	Proof              string          `json:"proof,omitempty"`
	LastError          *string         `json:"lastError"`
	Imports            []releaseImport `json:"imports"`
}

type releaseImport struct {
	ID            int64   `json:"id"`
	Tag           string  `json:"tag"`
	Status        string  `json:"status"`
	Failure       *string `json:"failure"`
	VersionNumber *int    `json:"versionNumber"`
}

func (s *GitHubReleases) source(ctx context.Context, ownerID, workID uuid.UUID) (*releaseSource, error) {
	var source releaseSource
	var proof string
	var verifiedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		select source.repository, source.attachment, source.include_prereleases,
		       source.proof, source.verified_at, source.last_error
		  from extension_release_sources source
		  join works work on work.id = source.work_id
		 where source.work_id = $1 and work.owner_id = $2 and work.deleted_at is null
	`, workID, ownerID).Scan(&source.Repository, &source.Attachment, &source.IncludePrereleases, &proof, &verifiedAt, &source.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	source.Verified = verifiedAt != nil
	if !source.Verified {
		source.Proof = proof
	}
	source.Imports = []releaseImport{}
	rows, err := s.pool.Query(ctx, `
		select release_id, tag, status, failure, version_number
		  from extension_release_imports where work_id = $1
		 order by published_at desc, release_id desc limit 30
	`, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item releaseImport
		if err := rows.Scan(&item.ID, &item.Tag, &item.Status, &item.Failure, &item.VersionNumber); err != nil {
			return nil, err
		}
		source.Imports = append(source.Imports, item)
	}
	return &source, rows.Err()
}

func (s *GitHubReleases) Configure(ctx context.Context, ownerID, workID uuid.UUID, repository string, attachment *string, includePrereleases bool) error {
	parsed, err := githubRepository(repository)
	if err != nil {
		return releaseSourceError(err.Error())
	}
	if attachment != nil && (len(*attachment) < 1 || len(*attachment) > 255 || strings.ContainsAny(*attachment, "/\\")) {
		return releaseSourceError("Enter one attachment file name.")
	}
	proofBytes := make([]byte, 32)
	if _, err := rand.Read(proofBytes); err != nil {
		return err
	}
	proof := base64.RawURLEncoding.EncodeToString(proofBytes)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	workType, err := work.LockEditable(ctx, tx, ownerID, workID)
	if err != nil {
		return err
	}
	if workType != "extension" {
		return errWrongType
	}
	var lifecycle string
	if err := tx.QueryRow(ctx, `select lifecycle from works where id = $1`, workID).Scan(&lifecycle); err != nil {
		return err
	}
	if lifecycle != "published" {
		return releaseSourceError("Publish the extension before connecting its repository.")
	}
	if _, err := tx.Exec(ctx, `delete from extension_release_sources where work_id = $1`, workID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `insert into extension_release_sources (work_id, repository, proof, attachment, include_prereleases)
		values ($1, $2, $3, $4, $5)`, workID, parsed, proof, attachment, includePrereleases); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *GitHubReleases) Verify(ctx context.Context, ownerID, workID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	var repository, proof string
	var verifiedAt *time.Time
	err = tx.QueryRow(ctx, `select repository, proof, verified_at from extension_release_sources where work_id = $1 for update`, workID).Scan(&repository, &proof, &verifiedAt)
	if err != nil {
		return err
	}
	if verifiedAt != nil {
		return nil
	}
	found, err := s.github.proof(ctx, repository)
	if err != nil {
		return releaseSourceError("Illarin could not read .illarin-proof. Check that the file is public and try again.")
	}
	if found != proof {
		return releaseSourceError(".illarin-proof does not contain the code shown here.")
	}
	if _, err := tx.Exec(ctx, `update extension_release_sources set verified_at = now(), next_check_at = now(), last_error = null where work_id = $1`, workID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *GitHubReleases) Disconnect(ctx context.Context, ownerID, workID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from extension_release_sources where work_id = $1`, workID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *GitHubReleases) Retry(ctx context.Context, ownerID, workID uuid.UUID, releaseID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	var status, repository string
	var assetID *int64
	var attachment *string
	err = tx.QueryRow(ctx, `select item.status, item.asset_id, source.attachment, source.repository
		from extension_release_imports item join extension_release_sources source on source.work_id = item.work_id
		where item.work_id = $1 and item.release_id = $2 for update of item`, workID, releaseID).Scan(&status, &assetID, &attachment, &repository)
	if err != nil {
		return err
	}
	if status != "failed" && status != "held" {
		return releaseSourceError("This release is not waiting for a retry.")
	}
	if attachment != nil && assetID == nil {
		releases, err := s.github.releases(ctx, repository)
		if err != nil {
			return releaseSourceError("Illarin could not check the attachment on GitHub. Try again.")
		}
		for _, release := range releases {
			if release.ID == releaseID {
				for _, asset := range release.Assets {
					if asset.Name == *attachment {
						assetID = &asset.ID
						break
					}
				}
				break
			}
		}
		if assetID == nil {
			return releaseSourceError("This release still has no attachment named " + *attachment + ".")
		}
	}
	if status == "held" {
		drafted, err := s.uploads.works.UnpublishedChanges(ctx, tx, workID)
		if err != nil {
			return err
		}
		if drafted {
			return releaseSourceError("Publish or discard your unpublished edits, then resume this release.")
		}
	}
	if _, err := tx.Exec(ctx, `update extension_release_imports set status = 'queued', failure = null, asset_id = $3 where work_id = $1 and release_id = $2`, workID, releaseID, assetID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *GitHubReleases) CheckNext(ctx context.Context) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var workID uuid.UUID
	var repository string
	var attachment *string
	var includePrereleases bool
	var verifiedAt time.Time
	err = tx.QueryRow(ctx, `
		select source.work_id, source.repository, source.attachment, source.include_prereleases, source.verified_at
		  from extension_release_sources source join works work on work.id = source.work_id
		 where source.verified_at is not null and source.next_check_at <= now()
		   and work.deleted_at is null and work.taken_down_at is null
		 order by source.next_check_at for update of source skip locked limit 1
	`).Scan(&workID, &repository, &attachment, &includePrereleases, &verifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	releases, checkErr := s.github.releases(ctx, repository)
	if checkErr == nil {
		for _, release := range releases {
			if release.ID == 0 || release.Tag == "" || release.Draft || release.PublishedAt.IsZero() || !release.PublishedAt.After(verifiedAt) || (release.Prerelease && !includePrereleases) {
				continue
			}
			var assetID *int64
			var failure *string
			if attachment != nil {
				for _, asset := range release.Assets {
					if asset.Name == *attachment {
						assetID = &asset.ID
						break
					}
				}
				if assetID == nil {
					message := "This release has no attachment named " + *attachment + "."
					failure = &message
				}
			}
			status := "queued"
			if failure != nil {
				status = "failed"
			}
			if _, err := tx.Exec(ctx, `insert into extension_release_imports (work_id, release_id, tag, published_at, asset_id, status, failure)
				values ($1, $2, $3, $4, $5, $6, $7) on conflict do nothing`, workID, release.ID, release.Tag, release.PublishedAt, assetID, status, failure); err != nil {
				return true, err
			}
		}
	}
	var message *string
	if checkErr != nil {
		text := checkErr.Error()
		message = &text
	}
	if _, err := tx.Exec(ctx, `update extension_release_sources set next_check_at = now() + interval '1 hour', last_error = $2 where work_id = $1`, workID, message); err != nil {
		return true, err
	}
	return true, tx.Commit(ctx)
}

func (s *GitHubReleases) ProcessNext(ctx context.Context) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var workID, ownerID uuid.UUID
	var releaseID, assetID int64
	var tag, repository, filename, workName string
	var attachment *string
	err = tx.QueryRow(ctx, `
		select item.work_id, work.owner_id, work.name, item.release_id, item.tag, source.repository,
		       coalesce(item.asset_id, 0), source.attachment
		  from extension_release_imports item
		  join extension_release_sources source on source.work_id = item.work_id
		  join works work on work.id = item.work_id
		 where item.status = 'queued' and work.deleted_at is null and work.taken_down_at is null
		   and not exists (
		       select 1 from extension_release_imports earlier
		        where earlier.work_id = item.work_id and earlier.status = 'held'
		          and (earlier.published_at, earlier.release_id) < (item.published_at, item.release_id)
		   )
		 order by item.published_at, item.release_id
		 for update of work, item skip locked limit 1
	`).Scan(&workID, &ownerID, &workName, &releaseID, &tag, &repository, &assetID, &attachment)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if attachment != nil {
		filename = *attachment
	}
	var archive []byte
	var fetchedName string
	workType, importErr := work.LockEditable(ctx, tx, ownerID, workID)
	if importErr == nil && workType != "extension" {
		importErr = errWrongType
	}
	if importErr == nil {
		var drafted bool
		drafted, importErr = s.uploads.works.UnpublishedChanges(ctx, tx, workID)
		if drafted {
			importErr = errGitHubImportHeld
		}
	}
	if importErr == nil {
		if attachment != nil && assetID == 0 {
			importErr = fmt.Errorf("this release has no attachment named %s", *attachment)
		} else {
			archive, fetchedName, importErr = s.github.archive(ctx, repository, tag, assetID)
		}
	}
	if filename == "" {
		filename = fetchedName
	}
	if importErr == nil {
		if _, err := tx.Exec(ctx, `savepoint before_release_import`); err != nil {
			return true, err
		}
		var published version.Version
		published, importErr = s.uploads.importGitHubRelease(ctx, tx, s.versions, ownerID, workID, filename, tag, archive)
		if importErr == nil {
			if _, err := tx.Exec(ctx, `update extension_release_imports set status = 'published', failure = null, version_number = $3
				where work_id = $1 and release_id = $2`, workID, releaseID, published.Number); err != nil {
				return true, err
			}
			return true, tx.Commit(ctx)
		}
		if _, err := tx.Exec(ctx, `rollback to savepoint before_release_import`); err != nil {
			return true, err
		}
	}
	status := "failed"
	message := importErr.Error()
	if errors.Is(importErr, errGitHubImportHeld) {
		status = "held"
		message = "Unpublished edits are waiting. Publish or discard them, then resume this release."
		if err := notify.Record(ctx, tx, notify.Event{Type: notify.GitHubReleaseHeld, Account: &ownerID, Work: &workID,
			Words: notify.Words{WorkName: workName}}); err != nil {
			return true, err
		}
	} else if _, _, classified := format.Explain(importErr); !classified {
		message = "Illarin could not import this release. Retry it after checking the archive."
	}
	if _, err := tx.Exec(ctx, `update extension_release_imports set status = $3, failure = $4 where work_id = $1 and release_id = $2`, workID, releaseID, status, message); err != nil {
		return true, err
	}
	return true, tx.Commit(ctx)
}

func (s *GitHubReleases) Run(ctx context.Context, report func(error)) {
	for {
		checked, err := s.CheckNext(ctx)
		if err == nil {
			var processed bool
			processed, err = s.ProcessNext(ctx)
			checked = checked || processed
		}
		if err != nil && ctx.Err() == nil && report != nil {
			report(err)
		}
		if ctx.Err() != nil {
			return
		}
		if checked && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func RegisterGitHubReleases(routes api.Routes, s *GitHubReleases) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/github-releases", d.JSON, s.getSource)
	routes.Handle(http.MethodPut, "/v1/works/:id/github-releases", d.JSON, s.configureSource)
	routes.Handle(http.MethodPost, "/v1/works/:id/github-releases/verify", d.JSON, s.verifySource)
	routes.Handle(http.MethodDelete, "/v1/works/:id/github-releases", d.JSON, s.disconnectSource)
	routes.Handle(http.MethodPost, "/v1/works/:id/github-releases/:releaseId/retry", d.JSON, s.retryRelease)
}

func sourceIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	owner, ok := api.Verified(c, "editing this extension")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return owner.ID, id, true
}

func refuseReleaseSource(c *gin.Context, err error, fallback string) {
	var visible releaseSourceError
	if errors.As(err, &visible) {
		api.Refuse(c, 422, visible.Error())
		return
	}
	if errors.Is(err, work.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		api.Refuse(c, 404, "This extension or release is not available.")
		return
	}
	if errors.Is(err, work.ErrWorkFrozen) {
		api.Refuse(c, 409, "This extension is unavailable while it is taken down.")
		return
	}
	if errors.Is(err, errWrongType) {
		api.Refuse(c, 422, "Only extensions can import GitHub releases.")
		return
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		api.Refuse(c, 409, "This repository is already connected to another extension.")
		return
	}
	api.Refuse(c, 500, fallback)
}

func (s *GitHubReleases) getSource(c *gin.Context) {
	owner, id, ok := sourceIDs(c)
	if !ok {
		return
	}
	source, err := s.source(c.Request.Context(), owner, id)
	if err != nil {
		api.Refuse(c, 500, "Illarin could not load GitHub releases. Try again.")
		return
	}
	c.JSON(200, source)
}

func (s *GitHubReleases) configureSource(c *gin.Context) {
	owner, id, ok := sourceIDs(c)
	if !ok {
		return
	}
	var body struct {
		Repository         string  `json:"repository"`
		Attachment         *string `json:"attachment"`
		IncludePrereleases bool    `json:"includePrereleases"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.Refuse(c, 400, "Enter a GitHub repository and release file choice.")
		return
	}
	if err := s.Configure(c.Request.Context(), owner, id, body.Repository, body.Attachment, body.IncludePrereleases); err != nil {
		refuseReleaseSource(c, err, "Illarin could not connect this repository. Try again.")
		return
	}
	s.getSource(c)
}

func (s *GitHubReleases) verifySource(c *gin.Context) {
	owner, id, ok := sourceIDs(c)
	if !ok {
		return
	}
	if err := s.Verify(c.Request.Context(), owner, id); err != nil {
		refuseReleaseSource(c, err, "Illarin could not verify this repository. Try again.")
		return
	}
	s.getSource(c)
}

func (s *GitHubReleases) disconnectSource(c *gin.Context) {
	owner, id, ok := sourceIDs(c)
	if !ok {
		return
	}
	if err := s.Disconnect(c.Request.Context(), owner, id); err != nil {
		api.Refuse(c, 404, "This extension is not available.")
		return
	}
	c.Status(204)
}

func (s *GitHubReleases) retryRelease(c *gin.Context) {
	owner, id, ok := sourceIDs(c)
	if !ok {
		return
	}
	var releaseID int64
	if _, err := fmt.Sscan(c.Param("releaseId"), &releaseID); err != nil || releaseID < 1 {
		api.Refuse(c, 400, "Choose a release to retry.")
		return
	}
	if err := s.Retry(c.Request.Context(), owner, id, releaseID); err != nil {
		refuseReleaseSource(c, err, "Illarin could not retry this release. Try again.")
		return
	}
	s.getSource(c)
}
