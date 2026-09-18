package upload_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func aBookOf(entries int) json.RawMessage {
	written := make([]string, entries)
	for i := range written {
		written[i] = `{"name":"Entry","keys":["ledger"],"secondaryKeys":["Mira"],` +
			`"selective":true,"enabled":` + map[bool]string{true: "true", false: "false"}[i > 0] +
			`,"order":100,"position":"before_character",` +
			`"recursion":{"prevent":true},"text":"Every debt is written in it."}`
	}
	return json.RawMessage(`{"entries":[` + strings.Join(written, ",") + `]}`)
}

func TestALorebookBlockSavesItsEntriesAndSaysHowManyItHolds(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	added := apitest.AddedBlock(t, apitest.AddBlock(t, r, session, started.ID, "lorebook", "entry_table"))
	if len(added.Elements) != 1 || added.Elements[0].Type != "entry_table" {
		t.Fatalf("a lorebook arrived holding %+v", added.Elements)
	}
	if added.Elements[0].Role != "lorebook_entries" {
		t.Fatalf("the book carries role %q", added.Elements[0].Role)
	}
	if added.Width != "two_thirds" || added.Layout != "single" {
		t.Errorf("a lorebook arrived %s wide in %s", added.Width, added.Layout)
	}

	body := apitest.EditableBlock(added)
	body.Elements[0].Content = aBookOf(1004)
	if response := apitest.SaveBlock(t, r, session, started.ID, added.ID, body); response.Code != http.StatusOK {
		t.Fatalf("save the book: status = %d: %s", response.Code, response.Body.String())
	}

	page := apitest.FetchStartedAsset(t, r, session, started.ID)
	var saved apitest.StartedBlock
	for _, holder := range page.Blocks {
		if holder.Definition == "lorebook" {
			saved = holder
		}
	}
	if saved.ID == "" {
		t.Fatalf("the lorebook is not on the page")
	}
	element := saved.Elements[0]
	if len(element.Facts) != 2 || element.Facts[0] != "1,004 entries" || element.Facts[1] != "1,003 switched on" {
		t.Errorf("the book says %v about itself", element.Facts)
	}
	var content struct {
		Entries []struct {
			Keys          []string `json:"keys"`
			SecondaryKeys []string `json:"secondaryKeys"`
			Selective     bool     `json:"selective"`
			Enabled       bool     `json:"enabled"`
			Position      string   `json:"position"`
			Recursion     struct {
				Prevent bool `json:"prevent"`
			} `json:"recursion"`
			Text string `json:"text"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(element.Content, &content); err != nil {
		t.Fatalf("read the saved book: %v", err)
	}
	if len(content.Entries) != 1004 {
		t.Fatalf("the saved book holds %d entries", len(content.Entries))
	}
	first := content.Entries[0]
	if first.Enabled || !first.Selective || !first.Recursion.Prevent ||
		first.Position != "before_character" || len(first.SecondaryKeys) != 1 {
		t.Errorf("the first entry came back as %+v", first)
	}
	if content.Entries[1].Text != "Every debt is written in it." {
		t.Errorf("entry 2 came back as %q", content.Entries[1].Text)
	}
}

func TestModelInstructionsArriveHoldingBothPrompts(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	added := apitest.AddedBlock(t, apitest.AddBlock(t, r, session, started.ID, "model_instructions", "prose"))
	if len(added.Elements) != 2 {
		t.Fatalf("model instructions arrived with %d elements", len(added.Elements))
	}
	roles := []string{added.Elements[0].Role, added.Elements[1].Role}
	if roles[0] != "system_prompt" || roles[1] != "post_history_instructions" {
		t.Errorf("model instructions carry roles %v", roles)
	}
	for _, element := range added.Elements {
		if element.Display != "verbatim" {
			t.Errorf("%s is shown as %q, want the exact text", element.Role, element.Display)
		}
	}
	if added.Layout != "stack-2" || added.Width != "half" {
		t.Errorf("model instructions arrived %s wide in %s", added.Width, added.Layout)
	}
}

func TestAnExpressionSetKeepsTheNamesItsSourceSupplied(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	added := apitest.AddedBlock(t, apitest.AddBlock(t, r, session, started.ID, "expressions", "image_set"))
	if added.Elements[0].Role != "expressions" {
		t.Fatalf("the expression set carries role %q", added.Elements[0].Role)
	}
	mediaID := apitest.UploadedImageID(t, r, session, started.ID, "expression", apitest.PNG(t, 200, 200))

	body := apitest.EditableBlock(added)
	body.Elements[0].Content = json.RawMessage(
		`{"images":[{"mediaId":"` + mediaID + `","name":"hey there. do you feel better now?"}]}`,
	)
	if response := apitest.SaveBlock(t, r, session, started.ID, added.ID, body); response.Code != http.StatusOK {
		t.Fatalf("save the expression set: status = %d: %s", response.Code, response.Body.String())
	}

	page := apitest.FetchStartedAsset(t, r, session, started.ID)
	for _, holder := range page.Blocks {
		if holder.Definition != "expressions" {
			continue
		}
		var content struct {
			Images []struct {
				Name string `json:"name"`
			} `json:"images"`
		}
		if err := json.Unmarshal(holder.Elements[0].Content, &content); err != nil {
			t.Fatalf("read the saved expression set: %v", err)
		}
		if content.Images[0].Name != "hey there. do you feel better now?" {
			t.Errorf("the expression name came back as %q", content.Images[0].Name)
		}
		if got := holder.Elements[0].Facts; len(got) != 1 || got[0] != "1 expression" {
			t.Errorf("the expression set says %v about itself", got)
		}
		return
	}
	t.Fatalf("the expression set is not on the page")
}
