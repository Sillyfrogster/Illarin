package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type postPicture struct {
	ID       string `json:"id"`
	PostID   string `json:"postId"`
	Purpose  string `json:"purpose"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type postHeader struct {
	MediaID string `json:"mediaId"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
}

func (s distinctionStack) upload(
	t *testing.T,
	session *http.Cookie,
	postID, purpose string,
	file []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	writeMetadataPart(t, form, map[string]any{"purpose": purpose})
	writeFilePartNamed(t, form, "picture.png", file)
	if err := form.Close(); err != nil {
		t.Fatalf("close picture form: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/v1/publication/posts/"+postID+"/media", &body,
	)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return send(t, s.router, authorized(request, session))
}

func (s distinctionStack) uploaded(
	t *testing.T,
	session *http.Cookie,
	postID, purpose string,
	file []byte,
) postPicture {
	t.Helper()
	response := s.upload(t, session, postID, purpose, file)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want 201: %s", response.Code, response.Body.String())
	}
	var picture postPicture
	if err := json.Unmarshal(response.Body.Bytes(), &picture); err != nil {
		t.Fatalf("decode picture: %v", err)
	}
	return picture
}

func bodyWithPicture(mediaID, alt string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"version":2,"content":[`+
			`{"type":"paragraph","content":[{"type":"text","text":"Here is what changed."}]},`+
			`{"type":"image","mediaId":%q,"alt":%q,"caption":"The workspace."}]}`,
		mediaID, alt,
	))
}

func (s distinctionStack) fetch(t *testing.T, address string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, httptest.NewRequest(http.MethodGet, address, nil))
}

func TestAnAuthorPlacesAnUploadedPictureAndAReaderReceivesIt(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "pictures@example.com", "illarin.pictures")
	draft := stack.illarinDraft(t, session, "The workspace has pictures")

	picture := stack.uploaded(t, session, draft.ID, "document", httpTestPNG(t, 1200, 600))
	if picture.PostID != draft.ID || picture.Purpose != "document" {
		t.Fatalf("picture = %+v, want it owned by the post as a document picture", picture)
	}
	if picture.Width != 1200 || picture.Height != 600 {
		t.Errorf("picture measured %dx%d, want 1200x600", picture.Width, picture.Height)
	}
	if !strings.Contains(picture.URL, "signature=") {
		t.Errorf("an unpublished picture is addressed as %q, want a signature on it", picture.URL)
	}
	plain := strings.SplitN(picture.URL, "?", 2)[0]
	if answered := stack.fetch(t, plain); answered.Code != http.StatusNotFound {
		t.Errorf("an unpublished picture answers %d without a signature", answered.Code)
	}
	if signed := stack.fetch(t, picture.URL); signed.Code != http.StatusOK {
		t.Fatalf("signed picture status = %d: %s", signed.Code, signed.Body.String())
	}

	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "The workspace with one draft in it"),
	}))
	if len(written.Media) != 1 || written.Media[0].ID != picture.ID {
		t.Fatalf("the working copy carries %+v, want the placed picture", written.Media)
	}

	stack.published(t, session, draft.ID)
	live := stack.reader(t, draft.Slug)
	if len(live.Media) != 1 || live.Media[0].ID != picture.ID {
		t.Fatalf("the published edition carries %+v, want the placed picture", live.Media)
	}
	if strings.Contains(live.Media[0].URL, "signature=") {
		t.Errorf("a published picture is addressed as %q, want no signature", live.Media[0].URL)
	}
	if answered := stack.fetch(t, live.Media[0].URL); answered.Code != http.StatusOK {
		t.Fatalf("published picture status = %d: %s", answered.Code, answered.Body.String())
	}
}

func TestAPostCannotPlaceAPictureAnotherPostOwns(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "foreign@example.com", "illarin.foreign")
	theirs := stack.illarinDraft(t, session, "The post that owns the picture")
	mine := stack.illarinDraft(t, session, "The post that wants it")

	picture := stack.uploaded(t, session, theirs.ID, "document", httpTestPNG(t, 400, 400))
	refused := stack.save(t, session, mine.ID, finished(mine, map[string]any{
		"document": bodyWithPicture(picture.ID, "A picture from somewhere else"),
	}))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("placing a foreign picture status = %d, want 400: %s",
			refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "does not belong to this post") {
		t.Errorf("refusal = %s", refused.Body.String())
	}
}

func TestAPictureIsPlacedOnlyWhereItWasUploadedFor(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "purpose@example.com", "illarin.purpose")
	draft := stack.illarinDraft(t, session, "One picture, one job")

	header := stack.uploaded(t, session, draft.ID, "header", httpTestPNG(t, 1600, 900))
	refused := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(header.ID, "The header, placed in the body"),
	}))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("placing a header in the body status = %d, want 400: %s",
			refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "uploaded as a header picture") {
		t.Errorf("refusal = %s", refused.Body.String())
	}
}

func TestEveryDisplayedPictureCarriesAlternativeText(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "alt@example.com", "illarin.alt")
	draft := stack.illarinDraft(t, session, "Pictures say what they show")

	picture := stack.uploaded(t, session, draft.ID, "document", httpTestPNG(t, 800, 400))
	refused := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "   "),
	}))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("an undescribed picture status = %d, want 400: %s",
			refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "document.content.1.alt") {
		t.Errorf("refusal does not name the picture: %s", refused.Body.String())
	}

	header := stack.uploaded(t, session, draft.ID, "header", httpTestPNG(t, 1600, 900))
	undescribed := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"header": map[string]any{"mediaId": header.ID, "alt": " "},
	}))
	if undescribed.Code != http.StatusBadRequest {
		t.Fatalf("an undescribed header status = %d, want 400: %s",
			undescribed.Code, undescribed.Body.String())
	}
	if !strings.Contains(undescribed.Body.String(), "header.alt") {
		t.Errorf("refusal does not name the header: %s", undescribed.Body.String())
	}
}

func TestReplacingAPictureLeavesTheOneAPublishedEditionCarries(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "replace@example.com", "illarin.replace")
	draft := stack.illarinDraft(t, session, "The picture that was replaced")

	first := stack.uploaded(t, session, draft.ID, "document", httpTestPNG(t, 900, 300))
	stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(first.ID, "The first picture"),
	}))
	stack.published(t, session, draft.ID)
	published := stack.reader(t, draft.Slug).Media[0].URL

	second := stack.uploaded(t, session, draft.ID, "document", httpTestPNG(t, 600, 600))
	if second.ID == first.ID {
		t.Fatal("replacing a picture reused the address readers already hold")
	}
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"version":  stack.reload(t, session, draft.ID).Version,
		"document": bodyWithPicture(second.ID, "The second picture"),
	}))
	if len(written.Media) != 1 || written.Media[0].ID != second.ID {
		t.Fatalf("the working copy carries %+v, want only the new picture", written.Media)
	}
	if answered := stack.fetch(t, published); answered.Code != http.StatusOK {
		t.Fatalf("the published picture answers %d after the working copy moved on", answered.Code)
	}
	if live := stack.reader(t, draft.Slug); live.Media[0].ID != first.ID {
		t.Errorf("the published edition now shows %s, want the picture it was published with",
			live.Media[0].ID)
	}
}

func TestAnUploadRefusesBytesThatAreNotAPicture(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "notapicture@example.com", "illarin.notapicture")
	draft := stack.illarinDraft(t, session, "Only pictures go here")

	refused := stack.upload(t, session, draft.ID, "document", []byte("this is not a picture"))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("uploading text status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
	unknown := stack.upload(t, session, draft.ID, "banner", httpTestPNG(t, 100, 100))
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("an unknown purpose status = %d, want 400: %s", unknown.Code, unknown.Body.String())
	}
}

func TestOnlyTheEditorOfAPostUploadsPicturesToIt(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "owner@example.com", "illarin.owner")
	outsider := stack.member(t, "outsider@example.com", "outsider.account")
	draft := stack.illarinDraft(t, session, "Not everyone writes here")

	refused := stack.upload(t, outsider, draft.ID, "document", httpTestPNG(t, 200, 200))
	if refused.Code != http.StatusForbidden {
		t.Fatalf("an outsider uploading status = %d, want 403: %s",
			refused.Code, refused.Body.String())
	}
}

func (s distinctionStack) reload(t *testing.T, session *http.Cookie, id string) blogPost {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+id, nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("reload post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}
