package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
)

type carriedImage struct {
	kind string
	name string
	data []byte
}

func TestASavedGalleryImageTravelsInEveryFormatThatCarriesIt(t *testing.T) {
	t.Parallel()
	r, session, assets := newCharacterIngestRouter(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	first, second := apitest.PNG(t, 64, 64), apitest.PNG(t, 48, 48)
	giveGallery(t, r, session, assetID, map[string][]byte{
		"At the door": first, "On the stair": second,
	})
	publishCharacter(t, r, session, assetID)

	for _, target := range downloadMenu(t, r, nil, assetID) {
		if roleVerdictNamed(t, target, "gallery").Verdict == "dropped" {
			continue
		}
		carried := imagesInDownload(t, r, assetID, target.Format)
		for name, wanted := range map[string][]byte{
			"At the door": first, "On the stair": second,
		} {
			if !carriesImage(carried, name, wanted) {
				t.Errorf("%s says it carries the gallery but %q is not in it: %+v",
					target.Format, name, describe(carried))
			}
		}
	}
}

func carriesImage(carried []carriedImage, name string, wanted []byte) bool {
	for _, image := range carried {
		if image.name == name && bytes.Equal(image.data, wanted) {
			return true
		}
	}
	return false
}

func describe(carried []carriedImage) []string {
	lines := make([]string, 0, len(carried))
	for _, image := range carried {
		lines = append(lines, image.kind+" "+image.name)
	}
	return lines
}

func giveGallery(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
	named map[string][]byte,
) {
	t.Helper()
	items := make([]string, 0, len(named))
	for name, file := range named {
		items = append(items, `{"mediaId":"`+
			uploadedImageID(t, r, session, assetID, "gallery", file)+`","name":"`+name+`"}`)
	}
	block := addedBlock(t, addBlock(t, r, session, assetID, "gallery", "image_set"))
	body := apitest.EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(`{"images":[` + strings.Join(items, ",") + `]}`)
	if saved := apitest.SaveBlock(t, r, session, assetID, block.ID, body); saved.Code != http.StatusOK {
		t.Fatalf("save the gallery: %d %s", saved.Code, saved.Body.String())
	}
}

func uploadedImageID(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID, role string,
	file []byte,
) string {
	t.Helper()
	added := apitest.Send(t, r, apitest.Authorized(mediaUploadRequest(t, assetID, role, file), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add a %s image: %d %s", role, added.Code, added.Body.String())
	}
	var picture struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(added.Body.Bytes(), &picture); err != nil {
		t.Fatalf("decode the added picture: %v", err)
	}
	return picture.ID
}

func imagesInDownload(t *testing.T, r http.Handler, assetID, target string) []carriedImage {
	t.Helper()
	return imagesInChosenDownload(t, r, assetID, target, nil)
}

func imagesInChosenDownload(
	t *testing.T,
	r http.Handler,
	assetID, target string,
	images *string,
) []carriedImage {
	t.Helper()
	address := "/download/" + assetID + "/" + target
	if images != nil {
		address += "?images=" + *images
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, address, nil))
	if download.Code != http.StatusOK {
		t.Fatalf("download %s: %d %s", target, download.Code, download.Body.String())
	}
	body := download.Body.Bytes()
	if target == "charx" {
		return archivedCardImages(t, body)
	}
	return inlineCardImages(t, body)
}

type writtenAsset struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
	Name string `json:"name"`
}

func writtenAssets(t *testing.T, card []byte) []writtenAsset {
	t.Helper()
	var document struct {
		Data struct {
			Assets []writtenAsset `json:"assets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(card, &document); err != nil {
		t.Fatalf("read the written card: %v", err)
	}
	return document.Data.Assets
}

func inlineCardImages(t *testing.T, card []byte) []carriedImage {
	t.Helper()
	carried := make([]carriedImage, 0)
	for _, record := range writtenAssets(t, card) {
		_, encoded, found := strings.Cut(record.URI, ";base64,")
		if !found {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("read the %s data URI: %v", record.Type, err)
		}
		carried = append(carried, carriedImage{kind: record.Type, name: record.Name, data: data})
	}
	return carried
}

func archivedCardImages(t *testing.T, archive []byte) []carriedImage {
	t.Helper()
	opened, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open the CharX: %v", err)
	}
	files := make(map[string][]byte, len(opened.File))
	var card []byte
	for _, entry := range opened.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatalf("open %s: %v", entry.Name, err)
		}
		var held bytes.Buffer
		if _, err := held.ReadFrom(reader); err != nil {
			t.Fatalf("read %s: %v", entry.Name, err)
		}
		reader.Close()
		if entry.Name == "card.json" {
			card = held.Bytes()
			continue
		}
		files[entry.Name] = held.Bytes()
	}
	carried := make([]carriedImage, 0, len(files))
	for _, record := range writtenAssets(t, card) {
		path, embedded := strings.CutPrefix(record.URI, "embeded://")
		if !embedded {
			continue
		}
		carried = append(carried, carriedImage{
			kind: record.Type, name: record.Name, data: files[path],
		})
	}
	return carried
}

func TestTheCreatorChoosesWhichGalleryImagesTravelByDefault(t *testing.T) {
	t.Parallel()
	r, session, assets := newCharacterIngestRouter(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	kept, left := apitest.PNG(t, 64, 64), apitest.PNG(t, 48, 48)
	gallery := savedGallery(t, r, session, assetID, []galleryItem{
		{name: "Kept", file: kept},
		{name: "Left out", file: left, omitted: true},
	})
	publishCharacter(t, r, session, assetID)

	carried := imagesInDownload(t, r, assetID, "charx")
	if !carriesImage(carried, "Kept", kept) {
		t.Errorf("the image the creator kept is missing: %+v", describe(carried))
	}
	if carriesImage(carried, "Left out", left) {
		t.Errorf("the image the creator left out travelled anyway: %+v", describe(carried))
	}
	if gallery["Left out"] == "" {
		t.Fatal("the left-out image was never stored")
	}
}

func TestAReaderChoosesImagesForOneDownloadAndChangesNothingStored(t *testing.T) {
	t.Parallel()
	r, session, assets := newCharacterIngestRouter(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	kept, left := apitest.PNG(t, 64, 64), apitest.PNG(t, 48, 48)
	gallery := savedGallery(t, r, session, assetID, []galleryItem{
		{name: "Kept", file: kept},
		{name: "Left out", file: left, omitted: true},
	})
	publishCharacter(t, r, session, assetID)

	wanted := gallery["Kept"] + "," + gallery["Left out"]
	both := imagesInChosenDownload(t, r, assetID, "charx", &wanted)
	if !carriesImage(both, "Kept", kept) || !carriesImage(both, "Left out", left) {
		t.Errorf("a reader asking for both got %+v", describe(both))
	}

	nothing := ""
	none := imagesInChosenDownload(t, r, assetID, "charx", &nothing)
	if len(none) != 0 {
		t.Errorf("a reader asking for no images got %+v", describe(none))
	}

	after := imagesInDownload(t, r, assetID, "charx")
	if carriesImage(after, "Left out", left) {
		t.Error("one reader's choice changed what the next download carries")
	}
	page := fetchStartedAsset(t, r, session, assetID)
	if !containsBytes(mustJSON(t, page.Blocks), []byte(`"omitFromDownloads":true`)) {
		t.Error("the creator's own choice is no longer on the asset")
	}
}

func TestADownloadRecordsItsFormatAndNothingAboutTheImagesChosen(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := newCharacterIngestRouterWithPool(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	gallery := savedGallery(t, r, session, assetID, []galleryItem{
		{name: "Kept", file: apitest.PNG(t, 64, 64)},
	})
	publishCharacter(t, r, session, assetID)

	chosen := gallery["Kept"]
	imagesInChosenDownload(t, r, assetID, "charx", &chosen)

	rows, err := pool.Query(context.Background(), `
		select export_target, authorization_class
		  from download_events where asset_id = $1
	`, assetID)
	if err != nil {
		t.Fatalf("read the download log: %v", err)
	}
	defer rows.Close()
	recorded := 0
	for rows.Next() {
		var target, class string
		if err := rows.Scan(&target, &class); err != nil {
			t.Fatalf("read a download event: %v", err)
		}
		if target != "charx" || class != "anonymous" {
			t.Errorf("event = %q by %q, want the format and a coarse class", target, class)
		}
		recorded++
	}
	if recorded != 1 {
		t.Fatalf("download events = %d, want the one handoff", recorded)
	}
	columns, err := pool.Query(context.Background(), `
		select column_name from information_schema.columns
		 where table_name = 'download_events'
	`)
	if err != nil {
		t.Fatalf("read the log's shape: %v", err)
	}
	defer columns.Close()
	for columns.Next() {
		var name string
		if err := columns.Scan(&name); err != nil {
			t.Fatalf("read a column: %v", err)
		}
		if strings.Contains(name, "image") || strings.Contains(name, "media") {
			t.Errorf("the download log keeps a column named %q", name)
		}
	}
}

func TestAnOversizedChoiceIsRefusedWholeRatherThanTrimmed(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := newCharacterIngestRouterWithPool(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	gallery := savedGallery(t, r, session, assetID, []galleryItem{
		{name: "Huge", file: apitest.PNG(t, 64, 64)},
		{name: "Small", file: apitest.PNG(t, 48, 48)},
	})
	publishCharacter(t, r, session, assetID)
	if _, err := pool.Exec(context.Background(), `
		update blobs set byte_size = $2
		 where id = (select blob_id from asset_media where id = $1)
	`, gallery["Huge"], int64(asset.MaxExportBytes)+1); err != nil {
		t.Fatalf("make one image oversized: %v", err)
	}

	refused := apitest.Send(t, r, httptest.NewRequest(
		http.MethodGet, "/download/"+assetID+"/charx", nil,
	))
	if refused.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want the whole selection refused: %s",
			refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "Leave some of them out") {
		t.Errorf("refusal = %s, want it to say what to do", refused.Body.String())
	}

	small := gallery["Small"]
	smaller := imagesInChosenDownload(t, r, assetID, "charx", &small)
	if len(smaller) != 1 {
		t.Fatalf("a smaller choice carried %+v", describe(smaller))
	}
}

func TestAnExpressionImageIsNotOfferedTheGallerysDownloadChoice(t *testing.T) {
	t.Parallel()
	r, session, assets := newCharacterIngestRouter(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	mediaID := uploadedImageID(t, r, session, assetID, "expression", apitest.PNG(t, 64, 64))

	block := addedBlock(t, addBlock(t, r, session, assetID, "expressions", "image_set"))
	body := apitest.EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(
		`{"images":[{"mediaId":"` + mediaID + `","name":"happy","omitFromDownloads":true}]}`,
	)
	saved := apitest.SaveBlock(t, r, session, assetID, block.ID, body)
	if saved.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want the choice refused where it has no meaning: %s",
			saved.Code, saved.Body.String())
	}
}

func TestTheOwnerAndAReaderAreToldTheSameAboutTheGallery(t *testing.T) {
	t.Parallel()
	r, session, assets := newCharacterIngestRouter(t)
	assetID := uploadedCharacterID(t, r, session, assets, aPlainCard)
	savedGallery(t, r, session, assetID, []galleryItem{
		{name: "Kept", file: apitest.PNG(t, 64, 64)},
		{name: "Left out", file: apitest.PNG(t, 48, 48), omitted: true},
	})
	publishCharacter(t, r, session, assetID)

	owner, reader := fetchAsset(t, r, session, assetID), fetchAsset(t, r, nil, assetID)
	if string(mustJSON(t, owner.Downloads)) != string(mustJSON(t, reader.Downloads)) {
		t.Errorf("the owner reads %s and a reader reads %s",
			mustJSON(t, owner.Downloads), mustJSON(t, reader.Downloads))
	}
	if string(mustJSON(t, owner.Blocks)) != string(mustJSON(t, reader.Blocks)) {
		t.Error("the owner and a reader are told different things about the gallery")
	}
}

type galleryItem struct {
	name    string
	file    []byte
	omitted bool
}

func savedGallery(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
	wanted []galleryItem,
) map[string]string {
	t.Helper()
	byName := make(map[string]string, len(wanted))
	items := make([]string, 0, len(wanted))
	for _, one := range wanted {
		mediaID := uploadedImageID(t, r, session, assetID, "gallery", one.file)
		byName[one.name] = mediaID
		choice := ""
		if one.omitted {
			choice = `,"omitFromDownloads":true`
		}
		items = append(items, `{"mediaId":"`+mediaID+`","name":"`+one.name+`"`+choice+`}`)
	}
	block := addedBlock(t, addBlock(t, r, session, assetID, "gallery", "image_set"))
	body := apitest.EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(`{"images":[` + strings.Join(items, ",") + `]}`)
	if saved := apitest.SaveBlock(t, r, session, assetID, block.ID, body); saved.Code != http.StatusOK {
		t.Fatalf("save the gallery: %d %s", saved.Code, saved.Body.String())
	}
	return byName
}

func fetchAsset(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
) apitest.StartedAsset {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	response := apitest.Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the asset: %d %s", response.Code, response.Body.String())
	}
	var page apitest.StartedAsset
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the asset: %v", err)
	}
	return page
}
