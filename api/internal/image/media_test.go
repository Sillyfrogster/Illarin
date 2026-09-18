package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreatorAddsMediaAndAnyoneFetchesAnImmutableVariant(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Theme with screenshots")
	metadata["_keepDraft"] = true
	metadata["filename"] = "theme.lumitheme"
	created := apitest.UploadAndFinish(t, r, session, works, metadata, []byte("theme"))
	workID := apitest.WorkIDFromIngest(t, created)

	added := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, workID, "gallery", apitest.PNG(t, 1200, 600),
	), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add media status = %d, want 201: %s", added.Code, added.Body.String())
	}
	var media struct {
		ID                string `json:"id"`
		WorkID            string `json:"assetId"`
		Role              string `json:"role"`
		Width             int    `json:"width"`
		Height            int    `json:"height"`
		DerivativeVersion uint32 `json:"derivativeVersion"`
	}
	if err := json.Unmarshal(added.Body.Bytes(), &media); err != nil {
		t.Fatalf("decode media response: %v", err)
	}
	if media.ID == "" || media.WorkID != workID {
		t.Fatalf("media response = %+v", media)
	}
	var fields map[string]any
	if err := json.Unmarshal(added.Body.Bytes(), &fields); err != nil {
		t.Fatalf("decode media fields: %v", err)
	}
	if _, exists := fields["revisionId"]; exists {
		t.Fatalf("media response still carries revisionId: %s", added.Body.String())
	}
	if media.Role != "gallery" || media.Width != 1200 || media.Height != 600 {
		t.Fatalf("media response = %+v", media)
	}
	if got := apitest.PublishWork(t, r, session, workID); got.Code != http.StatusOK {
		t.Fatalf("publish media: %d %s", got.Code, got.Body.String())
	}

	listed := apitest.Send(t, r, httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+workID+"/media", nil,
	))
	if listed.Code != http.StatusOK {
		t.Fatalf("list media status = %d, want 200: %s", listed.Code, listed.Body.String())
	}
	var mediaList struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &mediaList); err != nil {
		t.Fatalf("decode media list: %v", err)
	}
	if len(mediaList.Items) != 1 || mediaList.Items[0].ID != media.ID {
		t.Fatalf("media list = %+v, want %s", mediaList.Items, media.ID)
	}

	variantURL := "/media/" + media.ID + "/grid/" + "2"
	variant := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, variantURL, nil))
	if variant.Code != http.StatusOK {
		t.Fatalf("media status = %d, want 200: %s", variant.Code, variant.Body.String())
	}
	if got := variant.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := variant.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
	if got := variant.Header().Get("Content-Disposition"); got != "inline" {
		t.Errorf("Content-Disposition = %q, want inline", got)
	}
	if got := variant.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := variant.Header().Get("X-Accel-Redirect"); !strings.HasPrefix(got, "/_illarin/derivatives/") {
		t.Errorf("X-Accel-Redirect = %q, want internal derivative path", got)
	}
	if variant.Body.Len() != 0 {
		t.Errorf("Go wrote %d media bytes instead of handing off to nginx", variant.Body.Len())
	}
	preview := apitest.Send(t, r, httptest.NewRequest(
		http.MethodGet, "/media/"+media.ID+"/og/2", nil,
	))
	if preview.Code != http.StatusOK {
		t.Fatalf("og preview status = %d, want 200: %s", preview.Code, preview.Body.String())
	}
	if preview.Header().Get("X-Accel-Redirect") == variant.Header().Get("X-Accel-Redirect") {
		t.Fatal("composed og preview reused the grid derivative")
	}
	var eventCount int
	if err := pool.QueryRow(context.Background(), `select count(*) from download_events`).Scan(&eventCount); err != nil {
		t.Fatalf("count download events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("media handoffs wrote %d download events", eventCount)
	}
}

func TestMediaRouteRefusesArbitraryVariantsAndVersions(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)
	for _, path := range []string{
		"/media/11111111-1111-1111-1111-111111111111/1200x630/1",
		"/media/11111111-1111-1111-1111-111111111111/grid/999",
	} {
		response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", path, response.Code)
		}
	}
}

func TestMissingDerivativeYieldsToTheStorageReserveAndEvictsTheCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	r, session, works, pool := harness.NewVerifiedIngestRouterWithStoreFactory(
		t, format.NewRegistry(), work.DefaultIngestSettings(),
		func(pool *pgxpool.Pool) (storage.Store, error) {
			return storage.NewStore(pool, root)
		},
	)
	metadata := apitest.ExampleMetadata("Theme with one image")
	metadata["_keepDraft"] = true
	created := apitest.UploadAndFinish(
		t, r, session, works, metadata, []byte("theme"),
	)
	workID := apitest.WorkIDFromIngest(t, created)
	added := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, workID, "gallery", apitest.PNG(t, 120, 60),
	), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add media status = %d, want 201: %s", added.Code, added.Body.String())
	}
	var media struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(added.Body.Bytes(), &media); err != nil {
		t.Fatalf("decode media: %v", err)
	}

	if got := apitest.PublishWork(t, r, session, workID); got.Code != http.StatusOK {
		t.Fatalf("publish media: %d", got.Code)
	}

	unlimited, err := storage.NewStore(pool, root)
	if err != nil {
		t.Fatalf("open unlimited store: %v", err)
	}
	if err := unlimited.ClearDerivatives(context.Background()); err != nil {
		t.Fatalf("clear generated derivatives: %v", err)
	}
	var digestBytes []byte
	if err := pool.QueryRow(context.Background(), `
		select blob.sha256
		  from work_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
	`, media.ID).Scan(&digestBytes); err != nil {
		t.Fatalf("read media digest: %v", err)
	}
	var digest [32]byte
	copy(digest[:], digestBytes)
	disposable := storage.DerivativeID{SourceDigest: digest, Variant: "old", Version: 1}
	if err := unlimited.PutDerivative(context.Background(), disposable, []byte("old cache")); err != nil {
		t.Fatalf("seed disposable derivative: %v", err)
	}

	limited, err := storage.NewStoreWithCapacity(pool, root, storage.Capacity{
		FreeSpaceReserveBytes: int64(^uint64(0) >> 1),
		MaximumBlobWriteBytes: 1,
	})
	if err != nil {
		t.Fatalf("open limited store: %v", err)
	}
	limitedWorks := work.NewServiceWithIngestSettings(
		pool, format.NewRegistry(), limited, work.DefaultIngestSettings(),
	)
	handlers := apitest.NewServicesOver(pool, limited, limitedWorks, &apitest.VerificationOutbox{}, nil)
	limitedRouter := harness.RegisterRouter(t, handlers, api.DefaultDeadlines())

	response := apitest.Send(t, limitedRouter, httptest.NewRequest(
		http.MethodGet, "/media/"+media.ID+"/grid/2", nil,
	))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("media status = %d, want 503: %s", response.Code, response.Body.String())
	}
	if _, err := limited.OpenDerivative(context.Background(), disposable); !errors.Is(err, storage.ErrDerivativeNotFound) {
		t.Fatalf("disposable derivative survived low space: %v", err)
	}
}

func TestCreatorMediaCannotTakeTheAccountPastItsStorageCap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	var blobs storage.Store
	r, session, works, pool := harness.NewVerifiedIngestRouterWithStoreFactory(
		t, format.NewRegistry(), work.DefaultIngestSettings(),
		func(pool *pgxpool.Pool) (storage.Store, error) {
			var err error
			blobs, err = storage.NewStore(pool, root)
			return blobs, err
		},
	)
	source := []byte("theme")
	created := apitest.UploadAndFinish(
		t, r, session, works, apitest.ExampleMetadata("Theme at its cap"), source,
	)
	workID := apitest.WorkIDFromIngest(t, created)
	mediaBytes := apitest.PNG(t, 120, 60)

	settings := work.DefaultIngestSettings()
	settings.AccountStorageCapBytes = int64(len(source) + len(mediaBytes) - 1)
	limitedWorks := work.NewServiceWithIngestSettings(
		pool, format.NewRegistry(), blobs, settings,
	)
	handlers := apitest.NewServicesOver(pool, blobs, limitedWorks, &apitest.VerificationOutbox{}, nil)
	limitedRouter := harness.RegisterRouter(t, handlers, api.DefaultDeadlines())

	response := apitest.Send(t, limitedRouter, apitest.Authorized(
		apitest.MediaUploadRequest(t, workID, "gallery", mediaBytes), session,
	))

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("add media status = %d, want 413: %s", response.Code, response.Body.String())
	}
	var mediaCount int
	if err := pool.QueryRow(context.Background(), `select count(*) from work_media`).Scan(&mediaCount); err != nil {
		t.Fatalf("count media: %v", err)
	}
	if mediaCount != 0 {
		t.Fatalf("over-cap upload recorded %d media rows", mediaCount)
	}
}
