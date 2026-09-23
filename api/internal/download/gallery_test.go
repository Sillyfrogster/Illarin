package download_test

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
	"github.com/Sillyfrogster/Illarin/api/internal/download"
)

type carriedImage struct {
	workType string
	name     string
	data     []byte
}

func TestASavedGalleryImageTravelsInEveryFormatThatCarriesIt(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterUploadRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	first, second := apitest.PNG(t, 64, 64), apitest.PNG(t, 48, 48)
	giveGallery(t, r, session, workID, map[string][]byte{
		"At the door": first, "On the stair": second,
	})
	apitest.PublishCharacter(t, r, session, workID)

	for _, choice := range apitest.DownloadMenu(t, r, nil, workID) {
		if roleVerdictNamed(t, choice, "gallery").Verdict == "dropped" {
			continue
		}
		carried := imagesInDownload(t, r, workID, choice.Format)
		for name, wanted := range map[string][]byte{
			"At the door": first, "On the stair": second,
		} {
			if !carriesImage(carried, name, wanted) {
				t.Errorf("%s says it carries the gallery but %q is not in it: %+v",
					choice.Format, name, describe(carried))
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
		lines = append(lines, image.workType+" "+image.name)
	}
	return lines
}

func giveGallery(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
	named map[string][]byte,
) {
	t.Helper()
	items := make([]string, 0, len(named))
	for name, file := range named {
		items = append(items, `{"mediaId":"`+
			apitest.UploadedImageID(t, r, session, workID, "gallery", file)+`","name":"`+name+`"}`)
	}
	block := apitest.AddedBlock(t, apitest.AddBlock(t, r, session, workID, "gallery", "image_set"))
	body := apitest.EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(`{"images":[` + strings.Join(items, ",") + `]}`)
	if saved := apitest.SaveBlock(t, r, session, workID, block.ID, body); saved.Code != http.StatusOK {
		t.Fatalf("save the gallery: %d %s", saved.Code, saved.Body.String())
	}
}

func imagesInDownload(t *testing.T, r http.Handler, workID, formatID string) []carriedImage {
	t.Helper()
	return imagesInChosenDownload(t, r, workID, formatID, nil)
}

func imagesInChosenDownload(
	t *testing.T,
	r http.Handler,
	workID, formatID string,
	images *string,
) []carriedImage {
	t.Helper()
	address := "/download/" + workID + "/" + formatID
	if images != nil {
		address += "?images=" + *images
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, address, nil))
	if download.Code != http.StatusOK {
		t.Fatalf("download %s: %d %s", formatID, download.Code, download.Body.String())
	}
	body := download.Body.Bytes()
	if formatID == "charx" {
		return archivedCardImages(t, body)
	}
	return inlineCardImages(t, body)
}

type writtenWork struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
	Name string `json:"name"`
}

func writtenWorks(t *testing.T, card []byte) []writtenWork {
	t.Helper()
	var document struct {
		Data struct {
			Works []writtenWork `json:"assets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(card, &document); err != nil {
		t.Fatalf("read the written card: %v", err)
	}
	return document.Data.Works
}

func inlineCardImages(t *testing.T, card []byte) []carriedImage {
	t.Helper()
	carried := make([]carriedImage, 0)
	for _, record := range writtenWorks(t, card) {
		_, encoded, found := strings.Cut(record.URI, ";base64,")
		if !found {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("read the %s data URI: %v", record.Type, err)
		}
		carried = append(carried, carriedImage{workType: record.Type, name: record.Name, data: data})
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
	for _, record := range writtenWorks(t, card) {
		path, embedded := strings.CutPrefix(record.URI, "embeded://")
		if !embedded {
			continue
		}
		carried = append(carried, carriedImage{
			workType: record.Type, name: record.Name, data: files[path],
		})
	}
	return carried
}

func TestTheCreatorChoosesWhichGalleryImagesTravelByDefault(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterUploadRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	kept, left := apitest.PNG(t, 64, 64), apitest.PNG(t, 48, 48)
	gallery := savedGallery(t, r, session, workID, []galleryItem{
		{name: "Kept", file: kept},
		{name: "Left out", file: left, omitted: true},
	})
	apitest.PublishCharacter(t, r, session, workID)

	carried := imagesInDownload(t, r, workID, "charx")
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
	r, session, works := harness.NewCharacterUploadRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	kept, left := apitest.PNG(t, 64, 64), apitest.PNG(t, 48, 48)
	gallery := savedGallery(t, r, session, workID, []galleryItem{
		{name: "Kept", file: kept},
		{name: "Left out", file: left, omitted: true},
	})
	apitest.PublishCharacter(t, r, session, workID)

	wanted := gallery["Kept"] + "," + gallery["Left out"]
	both := imagesInChosenDownload(t, r, workID, "charx", &wanted)
	if !carriesImage(both, "Kept", kept) || !carriesImage(both, "Left out", left) {
		t.Errorf("a reader asking for both got %+v", describe(both))
	}

	nothing := ""
	none := imagesInChosenDownload(t, r, workID, "charx", &nothing)
	if len(none) != 0 {
		t.Errorf("a reader asking for no images got %+v", describe(none))
	}

	after := imagesInDownload(t, r, workID, "charx")
	if carriesImage(after, "Left out", left) {
		t.Error("one reader's choice changed what the next download carries")
	}
	page := apitest.FetchStartedWork(t, r, session, workID)
	if !apitest.ContainsBytes(mustJSON(t, page.Blocks), []byte(`"omitFromDownloads":true`)) {
		t.Error("the creator's own choice is no longer on the work")
	}
}

func TestADownloadRecordsItsFormatAndNothingAboutTheImagesChosen(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewCharacterUploadRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	gallery := savedGallery(t, r, session, workID, []galleryItem{
		{name: "Kept", file: apitest.PNG(t, 64, 64)},
	})
	apitest.PublishCharacter(t, r, session, workID)

	chosen := gallery["Kept"]
	imagesInChosenDownload(t, r, workID, "charx", &chosen)

	rows, err := pool.Query(context.Background(), `
		select format, access
		  from download_records where work_id = $1
	`, workID)
	if err != nil {
		t.Fatalf("read the download log: %v", err)
	}
	defer rows.Close()
	recorded := 0
	for rows.Next() {
		var formatID, access string
		if err := rows.Scan(&formatID, &access); err != nil {
			t.Fatalf("read a download record: %v", err)
		}
		if formatID != "charx" || access != "public" {
			t.Errorf("record = %q by %q, want the format and a coarse access", formatID, access)
		}
		recorded++
	}
	if recorded != 1 {
		t.Fatalf("download records = %d, want the one handoff", recorded)
	}
	columns, err := pool.Query(context.Background(), `
		select column_name from information_schema.columns
		 where table_name = 'download_records'
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
	r, session, works, pool := harness.NewCharacterUploadRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	gallery := savedGallery(t, r, session, workID, []galleryItem{
		{name: "Huge", file: apitest.PNG(t, 64, 64)},
		{name: "Small", file: apitest.PNG(t, 48, 48)},
	})
	apitest.PublishCharacter(t, r, session, workID)
	if _, err := pool.Exec(context.Background(), `
		update blobs set byte_size = $2
		 where id = (select blob_id from work_media where id = $1)
	`, gallery["Huge"], int64(download.MaxExportBytes)+1); err != nil {
		t.Fatalf("make one image oversized: %v", err)
	}

	refused := apitest.Send(t, r, httptest.NewRequest(
		http.MethodGet, "/download/"+workID+"/charx", nil,
	))
	if refused.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want the whole selection refused: %s",
			refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "Leave some of them out") {
		t.Errorf("refusal = %s, want it to say what to do", refused.Body.String())
	}

	small := gallery["Small"]
	smaller := imagesInChosenDownload(t, r, workID, "charx", &small)
	if len(smaller) != 1 {
		t.Fatalf("a smaller choice carried %+v", describe(smaller))
	}
}

func TestAnExpressionImageIsNotOfferedTheGallerysDownloadChoice(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterUploadRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	mediaID := apitest.UploadedImageID(t, r, session, workID, "expression", apitest.PNG(t, 64, 64))

	block := apitest.AddedBlock(t, apitest.AddBlock(t, r, session, workID, "expressions", "image_set"))
	body := apitest.EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(
		`{"images":[{"mediaId":"` + mediaID + `","name":"happy","omitFromDownloads":true}]}`,
	)
	saved := apitest.SaveBlock(t, r, session, workID, block.ID, body)
	if saved.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want the choice refused where it has no meaning: %s",
			saved.Code, saved.Body.String())
	}
}

func TestTheOwnerAndAReaderAreToldTheSameAboutTheGallery(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterUploadRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	savedGallery(t, r, session, workID, []galleryItem{
		{name: "Kept", file: apitest.PNG(t, 64, 64)},
		{name: "Left out", file: apitest.PNG(t, 48, 48), omitted: true},
	})
	apitest.PublishCharacter(t, r, session, workID)

	owner, reader := apitest.FetchWork(t, r, session, workID), apitest.FetchWork(t, r, nil, workID)
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
	workID string,
	wanted []galleryItem,
) map[string]string {
	t.Helper()
	byName := make(map[string]string, len(wanted))
	items := make([]string, 0, len(wanted))
	for _, one := range wanted {
		mediaID := apitest.UploadedImageID(t, r, session, workID, "gallery", one.file)
		byName[one.name] = mediaID
		choice := ""
		if one.omitted {
			choice = `,"omitFromDownloads":true`
		}
		items = append(items, `{"mediaId":"`+mediaID+`","name":"`+one.name+`"`+choice+`}`)
	}
	block := apitest.AddedBlock(t, apitest.AddBlock(t, r, session, workID, "gallery", "image_set"))
	body := apitest.EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(`{"images":[` + strings.Join(items, ",") + `]}`)
	if saved := apitest.SaveBlock(t, r, session, workID, block.ID, body); saved.Code != http.StatusOK {
		t.Fatalf("save the gallery: %d %s", saved.Code, saved.Body.String())
	}
	return byName
}
