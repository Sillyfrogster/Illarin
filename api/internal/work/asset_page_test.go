package work_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

func TestAssetPageCarriesItsCoverGalleryExpressionTagsAndBlurb(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("The Quiet Archivist")
	metadata["_keepDraft"] = true
	metadata["filename"] = "archivist.lumitheme"
	metadata["blurb"] = "She closes the book on a ribbon."
	metadata["tags"] = []string{"Slow Burn", " Modern "}
	assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, []byte("theme")))

	cover := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, assetID, "avatar", apitest.PNG(t, 800, 1000),
	), session))
	if cover.Code != http.StatusCreated {
		t.Fatalf("add cover status = %d, want 201: %s", cover.Code, cover.Body.String())
	}
	for _, role := range []string{"gallery", "expression"} {
		added := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
			t, assetID, role, apitest.PNG(t, 400, 300),
		), session))
		if added.Code != http.StatusCreated {
			t.Fatalf("add %s status = %d, want 201: %s", role, added.Code, added.Body.String())
		}
	}

	if got := apitest.PublishAsset(t, r, session, assetID); got.Code != http.StatusOK {
		t.Fatalf("publish media: %d %s", got.Code, got.Body.String())
	}

	page := apitest.FetchAssetPage(t, r, "/v1/assets/"+assetID)

	if page.ID != assetID || page.Name != "The Quiet Archivist" || page.Kind != "character" {
		t.Fatalf("asset page identity = %+v", page)
	}
	if page.Blurb != "She closes the book on a ribbon." {
		t.Errorf("blurb = %q, want the catalog blurb", page.Blurb)
	}
	if page.Creator != "verified.creator" {
		t.Errorf("creator = %q, want the owner's handle", page.Creator)
	}
	want := []struct{ label, value string }{{"Slow Burn", "slow burn"}, {" Modern ", "modern"}}
	if len(page.Tags) != len(want) {
		t.Fatalf("tags = %+v, want %d", page.Tags, len(want))
	}
	for i, tag := range want {
		if page.Tags[i].Label != tag.label || page.Tags[i].Value != tag.value {
			t.Errorf("tag %d = %+v, want %q shown and %q matched", i, page.Tags[i], tag.label, tag.value)
		}
	}
	if len(page.Media) != 3 {
		t.Fatalf("media = %+v, want the cover, gallery image and expression", page.Media)
	}
	image := page.Media[0]
	if image.Role != "avatar" || image.Width != 800 || image.Height != 1000 {
		t.Errorf("cover = %+v", image)
	}
	if !image.IsCover || page.Media[1].IsCover || page.Media[2].IsCover {
		t.Errorf("cover markers = %t, %t, %t", image.IsCover, page.Media[1].IsCover, page.Media[2].IsCover)
	}
	if image.DetailURL != "/media/"+image.ID+"/detail/2" {
		t.Errorf("detailUrl = %q", image.DetailURL)
	}
	if image.ThumbURL != "/media/"+image.ID+"/thumb/2" {
		t.Errorf("thumbUrl = %q", image.ThumbURL)
	}
	if page.Preview == nil || *page.Preview != "/media/"+image.ID+"/og/2" {
		t.Errorf("preview = %v, want the composed social preview", page.Preview)
	}
	if page.Media[1].Role != "gallery" || page.Media[2].Role != "expression" {
		t.Errorf("page media roles = %q, %q, %q", page.Media[0].Role, page.Media[1].Role, page.Media[2].Role)
	}
}

func TestAssetPageDoesNotPromoteGalleryMediaToCover(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Coverless Gallery")
	metadata["_keepDraft"] = true
	metadata["filename"] = "coverless-gallery.lumitheme"
	assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, []byte("theme")))

	added := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, assetID, "gallery", apitest.PNG(t, 400, 300),
	), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add gallery status = %d, want 201: %s", added.Code, added.Body.String())
	}

	if got := apitest.PublishAsset(t, r, session, assetID); got.Code != http.StatusOK {
		t.Fatalf("publish media: %d %s", got.Code, got.Body.String())
	}

	page := apitest.FetchAssetPage(t, r, "/v1/assets/"+assetID)
	if len(page.Media) != 1 || page.Media[0].Role != "gallery" || page.Media[0].IsCover {
		t.Fatalf("coverless gallery media = %+v", page.Media)
	}
}

func TestAssetPageShowsNoTotals(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Countless")
	metadata["filename"] = "countless.lumitheme"
	assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, []byte("theme")))

	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode asset page: %v", err)
	}

	wantKeys := map[string]bool{
		"id": true, "kind": true, "name": true, "blurb": true, "tags": true,
		"creator": true, "isNsfw": true, "discovery": true, "createdAt": true,
		"lifecycle": true, "isOwner": true, "downloads": true, "original": true,
		"appTargets": true,
		"blocks":     true, "media": true, "preview": true, "visibility": true,
		"linkedInstallOnly": true, "allowedApps": true, "eligibleApps": true,
		"latestUpdate": true, "extensionDependencies": true, "installedAppVersions": true,
	}
	for key := range body {
		if !wantKeys[key] {
			t.Errorf("asset page carries %q, which is not part of the page", key)
		}
	}
}

func TestAssetPageAnswersNormallyForAnUnlistedAsset(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Kept Back")
	metadata["filename"] = "kept-back.lumitheme"
	metadata["discovery"] = "unlisted"
	assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, []byte("theme")))

	page := apitest.FetchAssetPage(t, r, "/v1/assets/"+assetID)

	if page.Discovery != "unlisted" {
		t.Fatalf("discovery = %q, want unlisted", page.Discovery)
	}
}

func TestWithheldDeletedAndNeverExistedAssetsAnswerAlike(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	withhold := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(
		t, r, session, assets, withFilename(apitest.ExampleMetadata("Withheld"), "withheld"), []byte("a"),
	))
	deleted := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(
		t, r, session, assets, withFilename(apitest.ExampleMetadata("Deleted"), "deleted"), []byte("b"),
	))
	staff := "11111111-1111-1111-1111-111111111111"
	if _, err := pool.Exec(context.Background(), `
		insert into users (id, username) values ($1, 'staff')
	`, staff); err != nil {
		t.Fatalf("seed staff account: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		update assets
		   set withheld_at = now(), withheld_by = $2, withheld_reason = 'testing'
		 where id = $1
	`, withhold, staff); err != nil {
		t.Fatalf("withhold asset: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`update assets set deleted_at = now(), recoverable_until = now() + interval '30 days' where id = $1`, deleted,
	); err != nil {
		t.Fatalf("delete asset: %v", err)
	}

	never := "22222222-2222-2222-2222-222222222222"
	var bodies []string
	for _, id := range []string{withhold, deleted, never} {
		response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/assets/"+id, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET /v1/assets/%s status = %d, want 404: %s",
				id, response.Code, response.Body.String())
		}
		bodies = append(bodies, response.Body.String())
	}
	if bodies[0] != bodies[1] || bodies[1] != bodies[2] {
		t.Fatalf("the three refusals differ: %q", bodies)
	}
}

func TestBlurredReaderIsNeverHandedAClearVariant(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("After Dark")
	metadata["_keepDraft"] = true
	metadata["filename"] = "after-dark.lumitheme"
	metadata["isNsfw"] = true
	assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, []byte("theme")))
	added := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, assetID, "avatar", apitest.PNG(t, 600, 600),
	), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add media status = %d, want 201: %s", added.Code, added.Body.String())
	}

	if got := apitest.PublishAsset(t, r, session, assetID); got.Code != http.StatusOK {
		t.Fatalf("publish media: %d %s", got.Code, got.Body.String())
	}

	for _, preference := range []string{"blurred", "hidden"} {
		page := apitest.FetchAssetPage(t, r, "/v1/assets/"+assetID+"?nsfw="+preference)
		if len(page.Media) != 1 {
			t.Fatalf("%s media = %+v", preference, page.Media)
		}
		encoded, err := json.Marshal(page)
		if err != nil {
			t.Fatalf("re-encode asset page: %v", err)
		}
		for _, clear := range []string{"/detail/", "/thumb/", "/og/"} {
			if strings.Contains(string(encoded), clear) {
				t.Errorf("a %s reader was handed a %s variant: %s", preference, clear, encoded)
			}
		}
		if page.Preview == nil || !strings.Contains(*page.Preview, "/og_blurred/") {
			t.Errorf("%s preview = %v, want the blurred social preview", preference, page.Preview)
		}
	}

	shown := apitest.FetchAssetPage(t, r, "/v1/assets/"+assetID+"?nsfw=shown")
	if !strings.HasSuffix(shown.Media[0].DetailURL, "/detail/2") {
		t.Errorf("a shown reader got %q", shown.Media[0].DetailURL)
	}
	if shown.Preview == nil || !strings.Contains(*shown.Preview, "/og_blurred/") {
		t.Errorf("preview = %v; a link preview has no reader to ask, so it stays blurred", shown.Preview)
	}
}

func withFilename(metadata map[string]any, name string) map[string]any {
	metadata["filename"] = name + ".lumitheme"
	return metadata
}
