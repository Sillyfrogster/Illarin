package upload_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeRelease struct {
	ID         int64
	Tag        string
	Prerelease bool
	Published  time.Time
	Archive    []byte
}

// fakeRepository answers GitHub's proof, release list and archive requests for one repository
type fakeRepository struct {
	mu       sync.Mutex
	proof    string
	releases []fakeRelease
}

func (f *fakeRepository) client() *http.Client {
	return &http.Client{Transport: githubRoundTrip(func(request *http.Request) (*http.Response, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case strings.Contains(request.URL.Path, "/contents/.illarin-proof"):
			body, _ := json.Marshal(map[string]string{"type": "file", "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(f.proof))})
			return githubAnswer(body), nil
		case strings.HasSuffix(request.URL.Path, "/releases"):
			listed := make([]map[string]any, 0, len(f.releases))
			for _, release := range f.releases {
				listed = append(listed, map[string]any{"id": release.ID, "tag_name": release.Tag, "prerelease": release.Prerelease,
					"published_at": release.Published.UTC().Format(time.RFC3339), "assets": []any{}})
			}
			body, _ := json.Marshal(listed)
			return githubAnswer(body), nil
		case strings.Contains(request.URL.Path, "/zipball/"):
			for _, release := range f.releases {
				if strings.HasSuffix(request.URL.Path, "/zipball/"+release.Tag) {
					return githubAnswer(release.Archive), nil
				}
			}
		}
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Header: make(http.Header), Body: http.NoBody}, nil
	})}
}

func toolboxAt(t *testing.T, semver, script string) []byte {
	t.Helper()
	return apitest.ExtensionZip(t, map[string]string{
		"spindle.json":     strings.Replace(apitest.ToolboxManifest, "1.0.0", semver, 1),
		"dist/frontend.js": script,
	})
}

func startFixture(t *testing.T, repository *fakeRepository) (*upload.GitHubReleases, uuid.UUID, *pgxpool.Pool, http.Handler, *http.Cookie) {
	t.Helper()
	router, session, works, pool := harness.NewExtensionRouter(t)
	var owner uuid.UUID
	handle := apitest.SessionState(t, router, session).Handle
	if err := pool.QueryRow(t.Context(), `select id from users where username = $1`, handle).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	service := upload.NewGitHubReleases(upload.NewService(pool, works), version.NewService(pool, works), apitest.ProofKey, repository.client())
	return service, owner, pool, router, session
}

func TestGitHubStartDraftsTheLatestReleaseAndFollowsLaterOnes(t *testing.T) {
	t.Parallel()
	week := time.Now().Add(-7 * 24 * time.Hour)
	repository := &fakeRepository{releases: []fakeRelease{
		{ID: 1, Tag: "v1", Published: week, Archive: toolboxAt(t, "1.0.0", "one")},
		{ID: 2, Tag: "v2", Published: week.Add(time.Hour), Archive: toolboxAt(t, "2.0.0", "two")},
		{ID: 3, Tag: "v3-beta", Prerelease: true, Published: week.Add(2 * time.Hour), Archive: toolboxAt(t, "3.0.0-beta", "three")},
	}}
	service, owner, pool, router, session := startFixture(t, repository)
	address := "https://github.com/Example/Toolbox"
	proof, err := service.StartProof(owner, address)
	if err != nil {
		t.Fatal(err)
	}
	repository.proof = proof

	started, err := service.Start(t.Context(), owner, address, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	workID := started.ID
	var lifecycle, workType, workVersion string
	if err := pool.QueryRow(t.Context(), `select lifecycle, type, work_version from works where id = $1 and owner_id = $2`, workID, owner).
		Scan(&lifecycle, &workType, &workVersion); err != nil {
		t.Fatal(err)
	}
	if lifecycle != "draft" || workType != "extension" || workVersion != "2.0.0" {
		t.Fatalf("started work = %s %s %s, want a draft extension at the latest stable release", lifecycle, workType, workVersion)
	}

	repository.mu.Lock()
	repository.releases = append(repository.releases, fakeRelease{ID: 4, Tag: "v4", Published: time.Now(), Archive: toolboxAt(t, "4.0.0", "four")})
	repository.mu.Unlock()
	if _, err := service.CheckNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(t.Context(), `select status from extension_release_imports where work_id = $1 and release_id = 4`, workID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "held" {
		t.Fatalf("a release reaching an unpublished draft is %s, want held", status)
	}

	id := workID.String()
	if saved := apitest.SaveDetails(t, router, session, id, `{"name":"Quiet Toolbox","blurb":"","isNsfw":false}`); saved.Code != 204 {
		t.Fatalf("save details: %d %s", saved.Code, saved.Body.String())
	}
	if published := apitest.PublishWork(t, router, session, id); published.Code != 200 {
		t.Fatalf("publish: %d %s", published.Code, published.Body.String())
	}
	if err := service.Retry(t.Context(), owner, workID, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessNext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if number := apitest.VersionNumber(t, pool, id); number != 2 {
		t.Fatalf("version after the later release = %d, want 2", number)
	}
	var tags []string
	if err := pool.QueryRow(t.Context(), `select array_agg(tag order by release_id) from extension_release_imports where work_id = $1 and status = 'published'`, workID).Scan(&tags); err != nil {
		t.Fatal(err)
	}
	if strings.Join(tags, ",") != "v2,v4" {
		t.Fatalf("imported releases = %v, want v2 then v4", tags)
	}
}

func TestGitHubStartRefusesARepositoryTheAccountCannotProve(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{releases: []fakeRelease{
		{ID: 1, Tag: "v1", Published: time.Now().Add(-time.Hour), Archive: toolboxAt(t, "1.0.0", "one")},
	}}
	service, owner, pool, _, _ := startFixture(t, repository)
	address := "https://github.com/example/toolbox"
	someoneElse, err := service.StartProof(uuid.New(), address)
	if err != nil {
		t.Fatal(err)
	}
	repository.proof = someoneElse

	if _, err := service.Start(t.Context(), owner, address, nil, false); err == nil || !strings.Contains(err.Error(), ".illarin-proof") {
		t.Fatalf("start with another account's proof = %v, want a plain refusal naming the proof file", err)
	}
	var works int
	if err := pool.QueryRow(t.Context(), `select count(*) from works where owner_id = $1`, owner).Scan(&works); err != nil {
		t.Fatal(err)
	}
	if works != 0 {
		t.Fatalf("a refused start left %d works", works)
	}
}
