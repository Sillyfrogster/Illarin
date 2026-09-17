package blog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type importedPost struct {
	Post     blogPost     `json:"post"`
	Warnings []importNote `json:"warnings"`
}

type importRefusal struct {
	Error    string       `json:"error"`
	Code     string       `json:"code"`
	Field    string       `json:"field"`
	Refusals []importNote `json:"refusals"`
}

type importNote struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

const theSameThing = "## Release notes\n\n" +
	"Illarin now reads **Markdown**. See the [notes](https://example.com/notes).\n"

const theSameThingAsJSON = `{"version":2,"content":[
	{"type":"heading","level":2,"anchor":"release-notes",
	 "content":[{"type":"text","text":"Release notes"}]},
	{"type":"paragraph","content":[
		{"type":"text","text":"Illarin now reads "},
		{"type":"text","text":"Markdown","marks":[{"type":"bold"}]},
		{"type":"text","text":". See the "},
		{"type":"text","text":"notes",
		 "marks":[{"type":"link","href":"https://example.com/notes"}]},
		{"type":"text","text":"."}
	]}
]}`

func (s publicationStack) imported(
	t *testing.T,
	who contributor,
	id, markdown string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"version": s.working(t, who.session, id).Version, "markdown": markdown,
	})
	if err != nil {
		t.Fatalf("encode import: %v", err)
	}
	return s.sent(t, who, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/import", string(body),
	))
}

func decodeImport(t *testing.T, response *httptest.ResponseRecorder) importedPost {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
	var carried importedPost
	if err := json.Unmarshal(response.Body.Bytes(), &carried); err != nil {
		t.Fatalf("decode import: %v", err)
	}
	return carried
}

func decodeImportRefusal(t *testing.T, response *httptest.ResponseRecorder) importRefusal {
	t.Helper()
	var refused importRefusal
	if err := json.Unmarshal(response.Body.Bytes(), &refused); err != nil {
		t.Fatalf("decode refusal %q: %v", response.Body.String(), err)
	}
	return refused
}

func TestMarkdownAndJSONReachTheSameDocument(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	start := fmt.Sprintf(`{"categoryId":%q,"title":"Lumiverse 3 is out"}`, announcement.ID)

	written := stack.startedBy(t, writer, start)
	body, err := json.Marshal(finished(written, map[string]any{
		"document": json.RawMessage(theSameThingAsJSON),
	}))
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	saved := stack.sent(t, writer, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+written.ID, string(body),
	))
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d: %s", saved.Code, saved.Body.String())
	}

	converted := stack.startedBy(t, writer, start)
	carried := decodeImport(t, stack.imported(t, writer, converted.ID, theSameThing))

	if len(carried.Warnings) != 0 {
		t.Errorf("the import warned about %v", carried.Warnings)
	}
	if same(t, carried.Post.Document) != same(t, decodePost(t, saved).Document) {
		t.Errorf("Markdown became %s", same(t, carried.Post.Document))
	}
}

func TestAnImportSaysWhatItCouldNotCarryExactly(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")

	carried := decodeImport(t, stack.imported(t, writer, draft.ID,
		"One line  \nand the next.\n\n```brainfuck\n+++++.\n```\n"))

	if len(carried.Warnings) != 2 {
		t.Fatalf("the import warned %v", carried.Warnings)
	}
	if carried.Warnings[0].Line != 1 || carried.Warnings[1].Line != 4 {
		t.Errorf("the warnings are about lines %d and %d",
			carried.Warnings[0].Line, carried.Warnings[1].Line)
	}
	if len(carried.Post.Document.Content) != 2 {
		t.Errorf("the post carries %d blocks", len(carried.Post.Document.Content))
	}
}

func TestAnImportRefusesEveryThingItCannotCarryAtOnce(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")

	response := stack.imported(t, writer, draft.ID,
		"<div>One</div>\n\n![Photo](https://example.com/photo.png)\n\n[Click](javascript:alert)\n")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
	refused := decodeImportRefusal(t, response)
	if refused.Code != string(blog.PublicationErrorCodeInvalid) || refused.Field != "markdown" {
		t.Errorf("the refusal reads %+v", refused)
	}
	if len(refused.Refusals) != 3 {
		t.Fatalf("the refusal names %v", refused.Refusals)
	}
	for at, line := range []int{1, 3, 5} {
		if refused.Refusals[at].Line != line {
			t.Errorf("refusal %d is about line %d, want line %d",
				at, refused.Refusals[at].Line, line)
		}
		if refused.Refusals[at].Message == "" {
			t.Errorf("refusal %d says nothing about what to fix", at)
		}
	}
	if left := stack.working(t, writer.session, draft.ID); len(left.Document.Content) != 0 {
		t.Errorf("a refused import still changed the working copy to %v", left.Document)
	}
}

func TestAnImportCannotPlaceAnotherPostsPicture(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	mine := stack.draftBy(t, writer, "Lumiverse 3 is out")
	theirs := stack.draftBy(t, writer, "Lumiverse 2 is out")
	picture := stack.uploaded(t, writer.session, theirs.ID, "document", apitest.PNG(t, 900, 500))

	response := stack.imported(t, writer, mine.ID, fmt.Sprintf(
		"![A workspace](media:%s)\n", picture.ID,
	))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
	if refusalOf(t, response).Code != string(blog.PublicationErrorCodeInvalid) {
		t.Errorf("the refusal reads %s", response.Body.String())
	}
}

func TestAnImportPlacesAPictureThePostOwns(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")
	picture := stack.uploaded(t, writer.session, draft.ID, "document", apitest.PNG(t, 900, 500))

	carried := decodeImport(t, stack.imported(t, writer, draft.ID, fmt.Sprintf(
		"![A workspace](media:%s \"One draft.\")\n", picture.ID,
	)))
	placed := carried.Post.Document.Content
	if len(placed) != 1 || placed[0]["mediaId"] != picture.ID {
		t.Fatalf("the import placed %v", placed)
	}
	if placed[0]["alt"] != "A workspace" || placed[0]["caption"] != "One draft." {
		t.Errorf("the picture reads %v", placed[0])
	}
}

func TestAnImportBegunFromAnOlderVersionIsRefused(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")
	decodeImport(t, stack.imported(t, writer, draft.ID, theSameThing))

	stale, err := json.Marshal(map[string]any{
		"version": draft.Version, "markdown": theSameThing,
	})
	if err != nil {
		t.Fatalf("encode import: %v", err)
	}
	response := stack.sent(t, writer, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/import", string(stale),
	))
	if response.Code != http.StatusConflict {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
	if refused := refusalOf(t, response); refused.Code != string(blog.PublicationErrorCodeStaleVersion) ||
		refused.Version == nil {
		t.Errorf("the refusal reads %+v", refused)
	}
}

func TestAnImportKeepsNoMarkdownAnywhere(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")
	body, err := json.Marshal(map[string]any{
		"version": draft.Version, "markdown": theSameThing,
	})
	if err != nil {
		t.Fatalf("encode import: %v", err)
	}
	first := stack.sent(t, writer, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/import", string(body),
	))
	if first.Code != http.StatusOK {
		t.Fatalf("import status = %d: %s", first.Code, first.Body.String())
	}
	stack.sent(t, writer, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/revisions",
		fmt.Sprintf(`{"version":%d}`, draft.Version+1),
	))

	for _, kept := range []struct {
		what  string
		query string
	}{
		{"the post", `select document::text from posts where id = $1`},
		{"a revision", `select coalesce(string_agg(document::text, ' '), '')
		                  from post_revisions where post_id = $1`},
		{"the action log", `select coalesce(string_agg(
		                        coalesce(before_state, '') || ' ' || coalesce(after_state, ''), ' '
		                    ), '') from publication_audits where post_id = $1`},
	} {
		var held string
		err := stack.pool.QueryRow(context.Background(), kept.query, draft.ID).Scan(&held)
		if err != nil {
			t.Fatalf("read %s: %v", kept.what, err)
		}
		for _, mark := range []string{"## ", "**", "](https://example.com/notes)"} {
			if strings.Contains(held, mark) {
				t.Errorf("%s still carries %q", kept.what, mark)
			}
		}
	}
}

func TestOnlyTheImportOperationTakesMarkdown(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")
	body, err := json.Marshal(finished(draft, map[string]any{"document": theSameThing}))
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	response := stack.sent(t, writer, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("save status = %d: %s", response.Code, response.Body.String())
	}
}

func TestAContributorCannotImportIntoAnotherGrantsPost(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	mine := stack.contributor(t, "writer@example.com", "publication.writer")
	theirs := stack.contributor(t, "other@example.com", "publication.other")
	draft := stack.draftBy(t, mine, "Lumiverse 3 is out")

	body, err := json.Marshal(map[string]any{
		"version": draft.Version, "markdown": theSameThing,
	})
	if err != nil {
		t.Fatalf("encode import: %v", err)
	}
	response := stack.sent(t, theirs, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/import", string(body),
	))
	if response.Code != http.StatusForbidden {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
}

func TestAnImportLargerThanAPostIsRefusedBeforeItIsRead(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")
	body, err := json.Marshal(map[string]any{
		"version": draft.Version, "markdown": strings.Repeat("word ", 500000),
	})
	if err != nil {
		t.Fatalf("encode import: %v", err)
	}
	response := stack.sent(t, writer, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/import", string(body),
	))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
}

func TestMarkdownLongerThanAPostRefusesTheFieldItWasSentIn(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	draft := stack.draftBy(t, writer, "Lumiverse 3 is out")

	response := stack.imported(t, writer, draft.ID, strings.Repeat("word ", 60000))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("import status = %d: %s", response.Code, response.Body.String())
	}
	if refused := refusalOf(t, response); refused.Field != "markdown" {
		t.Errorf("the refusal reads %+v", refused)
	}
}

func (s publicationStack) draftBy(t *testing.T, who contributor, title string) blogPost {
	t.Helper()
	announcement := s.categoryBySlug(t, "announcement")
	return s.startedBy(t, who, fmt.Sprintf(
		`{"categoryId":%q,"title":%q}`, announcement.ID, title,
	))
}

func same(t *testing.T, document postDocument) string {
	t.Helper()
	out, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode document: %v", err)
	}
	return string(out)
}
