package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StartedWork struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Blurb     string `json:"blurb"`
	Lifecycle string `json:"lifecycle"`
	IsOwner   bool   `json:"isOwner"`
	IsNSFW    *bool  `json:"isNsfw"`
	Preview   *string
	Media     []struct {
		ID        string `json:"id"`
		DetailURL string `json:"detailUrl"`
		ThumbURL  string `json:"thumbUrl"`
	} `json:"media"`
	Readiness         []ReadinessItem  `json:"readiness"`
	Blocks            []StartedBlock   `json:"blocks"`
	AddableBlocks     []AddableBlock   `json:"addableBlocks"`
	Downloads         []DownloadFormat `json:"downloads"`
	AppFormats        []AppFormat      `json:"appFormats"`
	HasPrivatePrompts bool             `json:"hasPrivatePrompts"`
	AllowedApps       []AppName        `json:"allowedApps"`
	EligibleApps      []AppName        `json:"eligibleApps"`
	Original          *OriginalUpload  `json:"original"`
}

type StartedBlock struct {
	ID             string   `json:"id"`
	Definition     string   `json:"definition"`
	Title          string   `json:"title"`
	TitleIsDefault bool     `json:"titleIsDefault"`
	Position       int      `json:"position"`
	Hidden         bool     `json:"hidden"`
	Layout         string   `json:"layout"`
	Width          string   `json:"width"`
	AllowedLayouts []string `json:"allowedLayouts"`
	Required       bool     `json:"required"`
	Hideable       bool     `json:"hideable"`
	IsEmpty        bool     `json:"isEmpty"`
	Elements       []struct {
		ID       string          `json:"id"`
		Type     string          `json:"type"`
		Role     string          `json:"role"`
		Slot     string          `json:"slot"`
		Label    string          `json:"label"`
		Pinned   bool            `json:"pinned"`
		Display  string          `json:"display"`
		ItemSize string          `json:"itemSize"`
		IsEmpty  bool            `json:"isEmpty"`
		Facts    []string        `json:"facts"`
		Content  json.RawMessage `json:"content"`
	} `json:"elements"`
}

type AddableBlock struct {
	Definition string `json:"definition"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Group      string `json:"group"`
	GroupTitle string `json:"groupTitle"`
	Repeatable bool   `json:"repeatable"`
	Choices    []struct {
		Type  string `json:"type"`
		Label string `json:"label"`
	} `json:"choices"`
}

type DownloadFormat struct {
	Format      string        `json:"format"`
	Label       string        `json:"label"`
	Recommended bool          `json:"recommended"`
	Roles       []RoleVerdict `json:"roles"`
}

type AppName struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type AppFormat struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Format string `json:"format"`
}

type RoleVerdict struct {
	Role        string   `json:"role"`
	Label       string   `json:"label"`
	Verdict     string   `json:"verdict"`
	Reason      string   `json:"reason"`
	Destination string   `json:"destination"`
	ShownBy     []string `json:"shownBy"`
	Sample      struct {
		Count  int      `json:"count"`
		Texts  []string `json:"texts"`
		Images []string `json:"images"`
	} `json:"sample"`
}

type OriginalUpload struct {
	Label     string `json:"label"`
	MediaType string `json:"mediaType"`
	ArrivedAt string `json:"arrivedAt"`
}

func StartCharacter(t *testing.T, r http.Handler, session *http.Cookie) StartedWork {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/works",
		strings.NewReader(`{"type":"character"}`))
	request.Header.Set("Content-Type", "application/json")
	response := Send(t, r, Authorized(request, session))
	if response.Code != http.StatusCreated {
		t.Fatalf("start a character: status = %d, want 201: %s",
			response.Code, response.Body.String())
	}
	var started StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode the started asset: %v", err)
	}
	return started
}

func BlockNamed(t *testing.T, blocks []StartedBlock, definition string) StartedBlock {
	t.Helper()
	for _, b := range blocks {
		if b.Definition == definition {
			return b
		}
	}
	t.Fatalf("no %s block on the page", definition)
	return StartedBlock{}
}

type SaveBlockBody struct {
	Title             *string            `json:"title"`
	Layout            string             `json:"layout"`
	Width             string             `json:"width"`
	Elements          []SaveBlockElement `json:"elements"`
	AllowedApps       *[]string          `json:"allowedApps,omitempty"`
	MakePromptsPublic *bool              `json:"makePromptsPublic,omitempty"`
}

type SaveBlockElement struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Role     string          `json:"role,omitempty"`
	Slot     string          `json:"slot"`
	Display  string          `json:"display,omitempty"`
	ItemSize string          `json:"itemSize,omitempty"`
	Content  json.RawMessage `json:"content"`
}

func EditableBlock(block StartedBlock) SaveBlockBody {
	elements := make([]SaveBlockElement, len(block.Elements))
	for i, element := range block.Elements {
		elements[i] = SaveBlockElement{
			ID: element.ID, Type: element.Type, Role: element.Role,
			Slot: element.Slot, Display: element.Display, ItemSize: element.ItemSize,
			Content: element.Content,
		}
	}
	return SaveBlockBody{
		Layout:   block.Layout,
		Width:    block.Width,
		Elements: elements,
	}
}

func SaveBlock(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
	blockID string,
	body SaveBlockBody,
) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode block save: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPut,
		"/v1/works/"+workID+"/blocks/"+blockID,
		strings.NewReader(string(encoded)),
	)
	request.Header.Set("Content-Type", "application/json")
	return Send(t, r, Authorized(request, session))
}

type ReadinessItem struct {
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Detail  string  `json:"detail"`
	Met     bool    `json:"met"`
	BlockID *string `json:"blockId"`
}

func SaveDetails(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPut,
		"/v1/works/"+workID+"/details", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return Send(t, r, Authorized(request, session))
}

func WriteCharacterFloor(t *testing.T, r http.Handler, session *http.Cookie, started StartedWork) {
	t.Helper()
	if got := SaveDetails(t, r, session, started.ID,
		`{"name":"Ilse of the west shelf","blurb":"","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save details status = %d, want 204: %s", got.Code, got.Body.String())
	}
	coreBlock := BlockNamed(t, started.Blocks, "character_core")
	core := EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She keeps the books that forget themselves."}`)
	if got := SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save description status = %d, want 200: %s", got.Code, got.Body.String())
	}
	messagesBlock := BlockNamed(t, started.Blocks, "messages")
	messages := EditableBlock(messagesBlock)
	messages.Elements[0].Content = json.RawMessage(`{"texts":[{"text":"The west shelf moved again."}]}`)
	if got := SaveBlock(t, r, session, started.ID, messagesBlock.ID, messages); got.Code != http.StatusOK {
		t.Fatalf("save greeting status = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func PublishWork(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
) *httptest.ResponseRecorder {
	t.Helper()
	return Send(t, r, Authorized(
		httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/publish", nil), session))
}

func PublishedCharacter(t *testing.T, r *gin.Engine, session *http.Cookie) string {
	t.Helper()
	started := StartCharacter(t, r, session)
	WriteCharacterFloor(t, r, session, started)
	if published := PublishWork(t, r, session, started.ID); published.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", published.Code, published.Body.String())
	}
	return started.ID
}

func WithReviewedVersion(t *testing.T, r http.Handler, req *http.Request) {
	t.Helper()
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if req.Method == http.MethodGet || len(parts) < 4 || parts[0] != "v1" || parts[1] != "works" || req.Header.Get("X-Drafted-Changes-Version") != "" {
		return
	}
	switch parts[3] {
	case "details", "blocks", "publish", "versions", "preserved", "media", "original-file", "shelf", "found-images":
	default:
		return
	}
	read := httptest.NewRequest(http.MethodGet, "/v1/works/"+parts[2]+"?draftedChanges=true", nil)
	read.Header.Set("Cookie", req.Header.Get("Cookie"))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, read)
	var page struct {
		DraftedChangesVersion int64 `json:"draftedChangesVersion"`
	}
	if response.Code == http.StatusOK {
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
	}
	if page.DraftedChangesVersion == 0 {
		page.DraftedChangesVersion = 1
	}
	req.Header.Set("X-Drafted-Changes-Version", strconv.FormatInt(page.DraftedChangesVersion, 10))
}

// VersionNumber reads the number of the work's published version, which is 0 for a draft
func VersionNumber(t *testing.T, pool *pgxpool.Pool, workID string) int {
	t.Helper()
	id, err := uuid.Parse(workID)
	if err != nil {
		t.Fatalf("parse the work id: %v", err)
	}
	var number int
	if err := pool.QueryRow(t.Context(),
		`select coalesce(version.number, 0)
		   from works work
		   left join work_versions version on version.id = work.published_version_id
		  where work.id = $1`, id,
	).Scan(&number); err != nil {
		t.Fatalf("read the version number: %v", err)
	}
	return number
}

func StartPreset(t *testing.T, r http.Handler, session *http.Cookie, app string) StartedWork {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/works",
		strings.NewReader(`{"type":"preset","app":"`+app+`"}`))
	request.Header.Set("Content-Type", "application/json")
	response := Send(t, r, Authorized(request, session))
	if response.Code != http.StatusCreated {
		t.Fatalf("start a preset for %s: status = %d, want 201: %s",
			app, response.Code, response.Body.String())
	}
	var started StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode the started asset: %v", err)
	}
	return started
}

type PublishRefusal struct {
	Error     string          `json:"error"`
	Readiness []ReadinessItem `json:"readiness"`
}

func ItemNamed(t *testing.T, items []ReadinessItem, id string) ReadinessItem {
	t.Helper()
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no %s item in the readiness list %+v", id, items)
	return ReadinessItem{}
}

func PublishWorkVersion(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost,
		"/v1/works/"+workID+"/versions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return Send(t, r, Authorized(request, session))
}

func DownloadMenu(t *testing.T, r http.Handler, session *http.Cookie, workID string) []DownloadFormat {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil)
	if session != nil {
		request = Authorized(request, session)
	}
	response := Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the asset: status = %d: %s", response.Code, response.Body.String())
	}
	var page StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the asset: %v", err)
	}
	return page.Downloads
}
