package upload_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type githubRoundTrip func(*http.Request) (*http.Response, error)

func (f githubRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func githubAnswer(body []byte) *http.Response {
	return &http.Response{StatusCode: 200, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}
}

func githubFixture(t *testing.T, attachment string, archive []byte, proofOverride ...string) (*upload.GitHubReleases, string, uuid.UUID, *pgxpool.Pool, http.Handler, *http.Cookie) {
	t.Helper()
	router, session, works, pool := harness.NewExtensionRouter(t)
	first := apitest.ExtensionZip(t, map[string]string{"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "one"})
	if archive == nil {
		archive = first
	}
	workID := apitest.PublishExtension(t, router, session, works, "Quiet Toolbox", first)
	id := uuid.MustParse(workID)
	var owner uuid.UUID
	if err := pool.QueryRow(t.Context(), `select owner_id from works where id = $1`, id).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: githubRoundTrip(func(request *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(request.URL.Path, "/contents/.illarin-proof"):
			var proof string
			if err := pool.QueryRow(request.Context(), `select proof from extension_release_sources where work_id = $1`, id).Scan(&proof); err != nil {
				return nil, err
			}
			if len(proofOverride) > 0 {
				proof = proofOverride[0]
			}
			body, _ := json.Marshal(map[string]string{"type": "file", "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(proof))})
			return githubAnswer(body), nil
		case strings.HasSuffix(request.URL.Path, "/releases"):
			var assets []map[string]any
			if attachment != "" {
				assets = []map[string]any{{"id": 101, "name": attachment}}
			}
			body, _ := json.Marshal([]map[string]any{{"id": 51, "tag_name": "v2", "published_at": time.Now().UTC().Format(time.RFC3339), "assets": assets}})
			return githubAnswer(body), nil
		case strings.Contains(request.URL.Path, "/zipball/") || strings.Contains(request.URL.Path, "/releases/assets/101"):
			return githubAnswer(archive), nil
		default:
			return githubAnswer([]byte(`[]`)), nil
		}
	})}
	service := upload.NewGitHubReleases(upload.NewService(pool, works), version.NewService(pool, works), client)
	return service, workID, owner, pool, router, session
}

func TestGitHubRepositoryProofMustMatch(t *testing.T) {
	t.Parallel()
	service, workID, owner, _, _, _ := githubFixture(t, "", nil, "someone else's code")
	id := uuid.MustParse(workID)
	if err := service.Configure(t.Context(), owner, id, "https://github.com/example/toolbox", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Verify(t.Context(), owner, id); err == nil {
		t.Fatal("accepted the wrong repository proof")
	}
}

func TestGitHubReleaseMissingChosenAttachmentIsVisible(t *testing.T) {
	t.Parallel()
	service, workID, owner, pool, _, _ := githubFixture(t, "", nil)
	id := uuid.MustParse(workID)
	attachment := "extension.zip"
	if err := service.Configure(t.Context(), owner, id, "https://github.com/example/toolbox", &attachment, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Verify(t.Context(), owner, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `update extension_release_sources set verified_at = now() - interval '2 hours' where work_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CheckNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var status, failure string
	if err := pool.QueryRow(t.Context(), `select status, failure from extension_release_imports where work_id = $1`, id).Scan(&status, &failure); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || !strings.Contains(failure, attachment) || apitest.VersionNumber(t, pool, workID) != 1 {
		t.Fatalf("missing attachment = %s %q", status, failure)
	}
	if err := service.Retry(t.Context(), owner, id, 51); err == nil {
		t.Fatal("retried a release without its chosen attachment")
	}
	if processed, err := service.ProcessNext(t.Context()); err != nil || processed {
		t.Fatalf("missing attachment fell back to the source archive: %t %v", processed, err)
	}
}

func TestGitHubReleaseCannotPublishAnUnchangedArchive(t *testing.T) {
	t.Parallel()
	service, workID, owner, pool, _, _ := githubFixture(t, "", nil)
	id := uuid.MustParse(workID)
	if err := service.Configure(t.Context(), owner, id, "https://github.com/example/toolbox", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Verify(t.Context(), owner, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `update extension_release_sources set verified_at = now() - interval '2 hours' where work_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CheckNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(t.Context(), `select status from extension_release_imports where work_id = $1`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || apitest.VersionNumber(t, pool, workID) != 1 {
		t.Fatalf("unchanged release = %s; published version changed", status)
	}
}

func TestGitHubReleaseProofImportAndDuplicateDiscovery(t *testing.T) {
	t.Parallel()
	archive := apitest.ExtensionZip(t, map[string]string{
		"spindle.json":     strings.Replace(apitest.ToolboxManifest, "1.0.0", "2.0.0", 1),
		"dist/frontend.js": "two",
	})
	service, workID, owner, pool, router, _ := githubFixture(t, "extension.zip", archive)
	id := uuid.MustParse(workID)
	if err := service.Configure(t.Context(), uuid.New(), id, "https://github.com/example/toolbox", nil, false); err == nil {
		t.Fatal("an unrelated account connected the repository")
	}
	attachment := "extension.zip"
	if err := service.Configure(t.Context(), owner, id, "https://github.com/example/toolbox", &attachment, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Verify(t.Context(), owner, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `update extension_release_sources set verified_at = now() - interval '2 hours' where work_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if checked, err := service.CheckNext(t.Context()); err != nil || !checked {
		t.Fatalf("discover release: %t %v", checked, err)
	}
	if processed, err := service.ProcessNext(t.Context()); err != nil || !processed {
		t.Fatalf("import release: %t %v", processed, err)
	}
	if number := apitest.VersionNumber(t, pool, workID); number != 2 {
		t.Fatalf("version = %d, want 2", number)
	}
	download := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/"+extension.SpindleID, nil))
	if download.Code != 200 || !bytes.Equal(download.Body.Bytes(), archive) {
		t.Fatalf("imported download = %d, exact bytes = %t", download.Code, bytes.Equal(download.Body.Bytes(), archive))
	}
	if _, err := pool.Exec(t.Context(), `update extension_release_sources set next_check_at = now() where work_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CheckNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if processed, err := service.ProcessNext(t.Context()); err != nil || processed {
		t.Fatalf("duplicate release processed = %t: %v", processed, err)
	}
	if number := apitest.VersionNumber(t, pool, workID); number != 2 {
		t.Fatalf("duplicate moved version to %d", number)
	}
}

func TestGitHubReleaseHoldsUnpublishedEditsAndLeavesFailuresVisible(t *testing.T) {
	t.Parallel()
	archive := apitest.ExtensionZip(t, map[string]string{
		"spindle.json":     strings.Replace(apitest.ToolboxManifest, "1.0.0", "2.0.0", 1),
		"dist/frontend.js": "two",
	})
	service, workID, owner, pool, router, session := githubFixture(t, "", archive)
	id := uuid.MustParse(workID)
	if err := service.Configure(t.Context(), owner, id, "https://github.com/example/toolbox", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Verify(t.Context(), owner, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `update extension_release_sources set verified_at = now() - interval '2 hours' where work_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if saved := apitest.SaveDetails(t, router, session, workID, `{"name":"Edited locally","blurb":"","isNsfw":false}`); saved.Code != 204 {
		t.Fatalf("save edits: %d %s", saved.Code, saved.Body.String())
	}
	if _, err := service.CheckNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(t.Context(), `select status from extension_release_imports where work_id = $1`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "held" || apitest.VersionNumber(t, pool, workID) != 1 {
		t.Fatalf("status %s changed published version", status)
	}
	if err := service.Retry(t.Context(), owner, id, 51); err == nil {
		t.Fatal("resumed while edits still exist")
	}
	if _, err := pool.Exec(t.Context(), `update works set name = 'Quiet Toolbox' where id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if err := service.Retry(t.Context(), owner, id, 51); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if number := apitest.VersionNumber(t, pool, workID); number != 2 {
		t.Fatalf("resumed version = %d", number)
	}
}

func TestGitHubReleaseFailureKeepsPublishedBytes(t *testing.T) {
	t.Parallel()
	service, workID, owner, pool, router, session := githubFixture(t, "", []byte("not an archive"))
	id := uuid.MustParse(workID)
	if err := service.Configure(t.Context(), owner, id, "https://github.com/example/toolbox", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Verify(t.Context(), owner, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `update extension_release_sources set verified_at = now() - interval '2 hours' where work_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if saved := apitest.SaveDetails(t, router, session, workID, `{"name":"Edited locally","blurb":"","isNsfw":false}`); saved.Code != 204 {
		t.Fatalf("save edits: %d %s", saved.Code, saved.Body.String())
	}
	if _, err := service.CheckNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var status string
	var failure string
	if err := pool.QueryRow(t.Context(), `select status, failure from extension_release_imports where work_id = $1`, id).Scan(&status, &failure); err != nil {
		t.Fatal(err)
	}
	if status != "held" || apitest.VersionNumber(t, pool, workID) != 1 {
		t.Fatalf("invalid archive bypassed unpublished edits: %s", status)
	}
	if _, err := pool.Exec(t.Context(), `update works set name = 'Quiet Toolbox' where id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if err := service.Retry(t.Context(), owner, id, 51); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(t.Context(), `select status, failure from extension_release_imports where work_id = $1`, id).Scan(&status, &failure); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || failure == "" || apitest.VersionNumber(t, pool, workID) != 1 {
		t.Fatalf("failure = %s %q; published version changed", status, failure)
	}
	download := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/"+extension.SpindleID, nil))
	if download.Code != 200 || !bytes.Contains(download.Body.Bytes(), []byte("one")) {
		t.Fatalf("published archive changed: %d", download.Code)
	}
}
