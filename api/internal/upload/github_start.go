package upload

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Started names the extension draft a repository's release became
type Started struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// StartProof is the code an account commits as .illarin-proof to start an extension from a repository
func (s *GitHubReleases) StartProof(ownerID uuid.UUID, repository string) (string, error) {
	parsed, err := githubRepository(repository)
	if err != nil {
		return "", releaseSourceError(err.Error())
	}
	mac := hmac.New(sha256.New, s.proofKey)
	mac.Write([]byte("github-start\x00" + ownerID.String() + "\x00" + parsed))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Start imports the latest release of a repository the account proved as a new extension draft, and follows its later releases
func (s *GitHubReleases) Start(ctx context.Context, ownerID uuid.UUID, repository string, attachment *string, includePrereleases bool) (Started, error) {
	proof, err := s.StartProof(ownerID, repository)
	if err != nil {
		return Started{}, err
	}
	if err := checkAttachment(attachment); err != nil {
		return Started{}, err
	}
	parsed, _ := githubRepository(repository)
	found, err := s.github.proof(ctx, parsed)
	if err != nil {
		return Started{}, releaseSourceError("Illarin could not read .illarin-proof. Check that the file is public and try again.")
	}
	if !hmac.Equal([]byte(found), []byte(proof)) {
		return Started{}, releaseSourceError(".illarin-proof does not contain the code shown here.")
	}
	releases, err := s.github.releases(ctx, parsed)
	if err != nil {
		return Started{}, releaseSourceError("Illarin could not read the repository's releases on GitHub. Try again.")
	}
	var latest *githubRelease
	for i, release := range releases {
		if release.eligible(includePrereleases) && (latest == nil || release.PublishedAt.After(latest.PublishedAt)) {
			latest = &releases[i]
		}
	}
	if latest == nil && includePrereleases {
		return Started{}, releaseSourceError("This repository has no published releases.")
	}
	if latest == nil {
		return Started{}, releaseSourceError("This repository has no stable releases. Include prereleases, or publish a release on GitHub.")
	}
	var assetID int64
	if attachment != nil {
		id, found := latest.asset(*attachment)
		if !found {
			return Started{}, releaseSourceError("Release " + latest.Tag + " has no attachment named " + *attachment + ".")
		}
		assetID = id
	}
	archive, filename, err := s.github.archive(ctx, parsed, latest.Tag, assetID)
	if err != nil {
		return Started{}, releaseSourceError("Illarin could not download release " + latest.Tag + " from GitHub. Try again.")
	}
	if attachment != nil {
		filename = *attachment
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Started{}, err
	}
	defer tx.Rollback(ctx)
	started, err := s.uploads.startFromRelease(ctx, tx, ownerID, filename, archive)
	if err != nil {
		return Started{}, startRefusal(err)
	}
	if _, err := tx.Exec(ctx, `insert into extension_release_sources (work_id, repository, proof, attachment, include_prereleases, verified_at)
		values ($1, $2, $3, $4, $5, $6)`, started.ID, parsed, proof, attachment, includePrereleases, latest.PublishedAt); err != nil {
		return Started{}, err
	}
	var asset *int64
	if attachment != nil {
		asset = &assetID
	}
	if _, err := tx.Exec(ctx, `insert into extension_release_imports (work_id, release_id, tag, published_at, asset_id, status, version_number)
		values ($1, $2, $3, $4, $5, 'published', 1)`, started.ID, latest.ID, latest.Tag, latest.PublishedAt, asset); err != nil {
		return Started{}, err
	}
	return started, tx.Commit(ctx)
}

// startRefusal words why a release archive could not become a draft, keeping unexpected failures as they are
func startRefusal(err error) error {
	switch {
	case errors.Is(err, errWrongType), errors.Is(err, format.ErrUnsupportedFormat), errors.Is(err, ErrTypeNotBuildable):
		return releaseSourceError("This release's archive is not an extension Illarin can read.")
	case errors.Is(err, work.ErrStorageCap):
		return releaseSourceError("This release would take your account past its storage cap.")
	}
	if _, _, classified := format.Explain(err); classified {
		return releaseSourceError(refusal(err))
	}
	return err
}

func (s *GitHubReleases) startProof(c *gin.Context) {
	owner, ok := api.Verified(c, "starting an extension")
	if !ok {
		return
	}
	var body struct {
		Repository string `json:"repository"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.Refuse(c, 400, "Enter a GitHub repository.")
		return
	}
	proof, err := s.StartProof(owner.ID, body.Repository)
	if err != nil {
		refuseReleaseSource(c, err, "Illarin could not make a proof code. Try again.")
		return
	}
	c.JSON(200, gin.H{"proof": proof})
}

func (s *GitHubReleases) start(c *gin.Context) {
	owner, ok := api.Verified(c, "starting an extension")
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
	started, err := s.Start(c.Request.Context(), owner.ID, body.Repository, body.Attachment, body.IncludePrereleases)
	if err != nil {
		refuseReleaseSource(c, err, "Illarin could not start the extension. Try again.")
		return
	}
	c.Header("Location", "/v1/works/"+started.ID.String())
	c.JSON(http.StatusCreated, started)
}
