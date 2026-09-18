package preset

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
)

func TestSillyTavernExportsUseNativeMarkersAndPromptFlags(t *testing.T) {
	t.Parallel()
	list := block.PromptList{Fragments: []block.PromptFragment{
		{ID: block.NewItemID(), Marker: "worldInfoBefore", Enabled: true},
		{ID: block.NewItemID(), Marker: "chatHistory", Enabled: true},
		{ID: block.NewItemID(), Marker: "chatHistory", Enabled: false},
		{ID: block.NewItemID(), Marker: "enhanceDefinitions", Text: "Add detail."},
		{ID: block.NewItemID(), Marker: "nsfw", Text: "Keep the tone."},
		{ID: block.NewItemID(), Name: "Instructions", Role: block.PromptSystem, Text: "Be brief.", Enabled: true},
	}}
	parsed := format.Parsed{Elements: []block.Element{{
		Type: block.TypePromptList, Role: block.RolePromptFragments, Content: list,
	}}}
	for _, fragment := range list.Fragments {
		if fragment.Marker == "" {
			continue
		}
		parsed.Remainder = append(parsed.Remainder, format.Remainder{
			Owner: format.OwnerItem, OwnerID: fragment.ID, Namespace: sillyTavernPromptNamespace,
			Payload: keys.Must(map[string]any{"system_prompt": false}),
		})
	}
	var body struct {
		Prompts []struct {
			Identifier   string `json:"identifier"`
			Marker       bool   `json:"marker"`
			SystemPrompt *bool  `json:"system_prompt"`
		}
		Orders []struct {
			Order []struct {
				Identifier string `json:"identifier"`
				Enabled    bool   `json:"enabled"`
			} `json:"order"`
		} `json:"prompt_order"`
	}
	if err := json.Unmarshal(write(t, SillyTavernModule{}, parsed).Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Prompts) != 5 || len(body.Orders[0].Order) != 5 {
		t.Fatalf("export has %d prompts and %d order entries, want 5", len(body.Prompts), len(body.Orders[0].Order))
	}
	want := []string{"worldInfoBefore", "chatHistory", "enhanceDefinitions", "nsfw", list.Fragments[5].ID.String()}
	for i, prompt := range body.Prompts {
		if prompt.Identifier != want[i] || body.Orders[0].Order[i].Identifier != want[i] {
			t.Errorf("native prompt identifier = %q", prompt.Identifier)
		}
		if prompt.Marker != (i < 2) || prompt.SystemPrompt == nil || *prompt.SystemPrompt != (i < 4) {
			t.Errorf("wrong prompt flags for %q", prompt.Identifier)
		}
	}
	if !body.Orders[0].Order[1].Enabled {
		t.Error("the duplicate history marker disabled the active history")
	}
}

func TestTheSillyTavernSignatureIsDisjointFromTheThemes(t *testing.T) {
	t.Parallel()
	themeKeys := []string{"main_text_color", "blur_strength"}
	recognition := (SillyTavernModule{}).Declaration().Recognition
	if len(recognition) != 1 || recognition[0].Type != format.RecognitionSignature {
		t.Fatalf("recognition = %+v, want one structural signature", recognition)
	}
	required := recognition[0].Required
	if len(required) != 2 {
		t.Fatalf("required keys = %v, want the prompt list and its order", required)
	}
	for _, key := range themeKeys {
		if _, shared := required[key]; shared {
			t.Errorf("the preset signature requires %q, which is a theme's key", key)
		}
	}
	theme := document(t, `{"name":"Glimmer","main_text_color":"rgba(1,1,1,1)","blur_strength":8}`)
	if _, claimed := (SillyTavernModule{}).Claim(theme); claimed {
		t.Error("the preset module claimed a SillyTavern theme")
	}
}

func TestTheSillyTavernOrderDecidesTheFragmentsAndTheirSwitches(t *testing.T) {
	t.Parallel()
	parsed := parse(t, sillyTavernPreset)
	if parsed.Format != SillyTavernID {
		t.Fatalf("parsed format = %q", parsed.Format)
	}
	list := promptList(t, parsed.Elements)
	if len(list.Fragments) != 4 {
		t.Fatalf("read %d fragments, want 4", len(list.Fragments))
	}
	names := make([]string, 0, len(list.Fragments))
	for _, fragment := range list.Fragments {
		names = append(names, fragment.Name)
	}
	want := []string{"Author note", "| Prompt", "Chat History", "Not in the order"}
	if !slices.Equal(names, want) {
		t.Errorf("fragments came out as %v, want %v", names, want)
	}
	if list.Fragments[0].Enabled {
		t.Error("a fragment the order switches off came out switched on")
	}
	if !list.Fragments[1].Enabled || !list.Fragments[2].Enabled {
		t.Error("a fragment the order switches on came out switched off")
	}
	if list.Fragments[3].Enabled {
		t.Error("a fragment the order never names came out switched on")
	}
	if note := list.Fragments[0]; note.Placement != block.InHistory ||
		note.Depth == nil || *note.Depth != 2 {
		t.Errorf("the note = %+v, want it placed in the history at a depth", note)
	}
	if marker := list.Fragments[2]; marker.Marker != "chatHistory" {
		t.Errorf("marker = %q, want what the file holds a place for", marker.Marker)
	}

	body := preservedPayload(t, parsed.Remainder, format.OwnerWork, sillyTavernNamespace)
	var others []map[string]any
	if err := json.Unmarshal(body["prompt_order"], &others); err != nil {
		t.Fatalf("read the preserved order: %v", err)
	}
	if len(others) != 1 || others[0]["character_id"] != float64(100000) {
		t.Errorf("preserved orders = %+v, want the one this app does not read", others)
	}
	if _, held := body["topP"]; !held {
		t.Error("a name belonging to the other preset format was read rather than preserved")
	}
}
