package extension

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/google/uuid"
)

const spindleBackend = `
spindle.registerTool({ name: "search_notes", display_name: "Search notes", parameters: { type: "object" } })
spindle.registerMacro({ name: "weather", description: "The weather outside", handler: () => "sunny" })
spindle.commands.register([
	{ id: "summarize-chat", label: "Summarize Chat" },
	{ id: "export-notes", label: "Export Notes" },
])
spindle.registerInterceptor(async (messages) => messages, 50)
spindle.registerContextHandler(async (context) => context)
spindle.registerTool({ name: "search_notes" })
`

const spindleFrontend = `
export function setup(ctx) {
	ctx.ui.registerDrawerTab({ id: "notes", title: "Quiet Notes", shortName: "Notes" })
	ctx.ui.registerInputBarAction({ id: "tidy", label: "Tidy the reply" })
	ctx.ui.createFloatWidget({ width: 48, height: 48 })
}
`

func TestSpindleListsWhatItsCodeAddsToLumiverse(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json":     sampleManifest,
		"dist/backend.js":  spindleBackend,
		"dist/frontend.js": spindleFrontend,
	}))

	want := []string{
		"Commands | Summarize Chat",
		"Commands | Export Notes",
		"Macros | {{weather}}",
		"Tools | search_notes",
		"UI surfaces | Drawer tab: Quiet Notes",
		"UI surfaces | Input bar action: Tidy the reply",
		"UI surfaces | Float widget",
		"Generation hooks | Prompt interceptor",
		"Generation hooks | Context handler",
	}
	if got := additions(t, parsed); !slices.Equal(got, want) {
		t.Fatalf("additions = %q\nwant %q", got, want)
	}
}

func TestNamesBuiltWhileTheCodeRunsAreNotListed(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json": sampleManifest,
		"dist/backend.js": "spindle.registerTool({ name: toolName })\n" +
			"spindle.registerTool({ name: `tool_${suffix}` })\n" +
			"spindle.registerTool({ name: \"   \" })\n" +
			"spindle.registerMacro(definition)\n" +
			"spindle.commands.register(commands)\n" +
			"registry.registerTool({ name: \"someone_elses_tool\" })\n",
		"dist/frontend.js": `export function setup(ctx) { ctx.ui.registerDrawerTab({ title: labels.drawer }) }`,
	}))

	if got := additions(t, parsed); !slices.Equal(got, []string{"UI surfaces | Drawer tab"}) {
		t.Fatalf("additions = %q, want only the drawer tab, whose title is built while it runs", got)
	}
}

func TestAnExtensionWhoseCodeRegistersNothingListsNoAdditions(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json": sampleManifest, "dist/frontend.js": "export default {}",
	}))
	for _, element := range parsed.Elements {
		if element.Role == block.RoleExtensionAdditions {
			t.Fatalf("additions = %+v, want none", element.Content)
		}
	}
}

func TestSpindleFollowsTheImportsOfTheSourceLumiverseBuilds(t *testing.T) {
	manifest := strings.Replace(sampleManifest, `"entry_frontend": "dist/frontend.js",`, "", 1)
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json":        manifest,
		"src/backend.ts":      "import { registerTools } from \"./tools\"\nimport \"./macros/index.js\"\nregisterTools()\n",
		"src/tools.ts":        "export function registerTools(): void {\n\tspindle.registerTool({ name: \"from_a_module\" } as ToolDefinition)\n}\n",
		"src/macros/index.ts": `spindle.registerMacro({ name: "from_an_index" })`,
		"src/tools.test.ts":   `spindle.registerTool({ name: "from_a_test" })`,
		"scripts/build.ts":    `spindle.registerTool({ name: "from_the_build" })`,
	}))

	want := []string{"Macros | {{from_an_index}}", "Tools | from_a_module"}
	if got := additions(t, parsed); !slices.Equal(got, want) {
		t.Fatalf("additions = %q, want %q and nothing from tests or tooling", got, want)
	}
}

func TestSpindleReadsTheBuiltCodeRatherThanTheSourceBesideIt(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json":     sampleManifest,
		"dist/frontend.js": `export function setup(ctx) { ctx.ui.registerDrawerTab({ title: "Built" }) }`,
		"src/frontend.ts":  `export function setup(ctx) { ctx.ui.registerDrawerTab({ title: "Source" }) }`,
	}))

	if got := additions(t, parsed); !slices.Equal(got, []string{"UI surfaces | Drawer tab: Built"}) {
		t.Fatalf("additions = %q, want the built entry Lumiverse runs", got)
	}
}

const tavernEntry = `
import { rollCommands } from "./src/commands.js";
import { eventSource, event_types } from "../../../../script.js";
const { registerFunctionTool } = SillyTavern.getContext();
registerFunctionTool({ name: "RollTheDice", displayName: "Dice roll", action: roll });
eventSource.on(event_types.MESSAGE_RECEIVED, onMessage);
eventSource.once(event_types.APP_READY, init);
eventSource.on(event_types.MESSAGE_RECEIVED, onMessageAgain);
MacrosParser.registerMacro("roll_total", () => total);
SillyTavern.getContext().macros.register("last_roll", { handler: () => last });
rollCommands();
`

const tavernCommands = `
export function rollCommands() {
	SlashCommandParser.addCommandObject(SlashCommand.fromProps({ name: "roll", aliases: ["r"], callback: roll }));
	registerSlashCommand("rollquiet", rollQuiet, [], "Rolls without a message");
	SlashCommandParser.addCommandObject(SlashCommand.fromProps({ name: commandName }));
}
`

func TestSillyTavernListsWhatItsCodeAddsToSillyTavern(t *testing.T) {
	parsed := parseTavern(t, spindleZip(t, map[string]string{
		"manifest.json":   strings.Replace(tavernManifest, "dist/index.js", "index.js", 1),
		"index.js":        tavernEntry,
		"src/commands.js": tavernCommands,
	}))

	want := []string{
		"Commands | /roll",
		"Commands | /rollquiet",
		"Macros | {{roll_total}}",
		"Macros | {{last_roll}}",
		"Tools | RollTheDice",
		"Generation hooks | MESSAGE_RECEIVED",
		"Generation hooks | APP_READY",
	}
	if got := additions(t, parsed); !slices.Equal(got, want) {
		t.Fatalf("additions = %q\nwant %q", got, want)
	}
}

func TestTheCodeIsReadNoFurtherThanTheLargestArchiveHolds(t *testing.T) {
	half := MaxArchiveBytes/2 + 1<<20
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json":    sampleManifest,
		"dist/backend.js": "spindle.registerTool({ name: \"near_the_start\" })\n" + padding(half),
		"dist/frontend.js": "export function setup(ctx) {\n\tctx.ui.registerDrawerTab({ title: \"Within reach\" })\n" +
			padding(half) + "\tctx.ui.registerInputBarAction({ label: \"Past the limit\" })\n}\n",
	}))

	want := []string{"Tools | near_the_start", "UI surfaces | Drawer tab: Within reach"}
	if got := additions(t, parsed); !slices.Equal(got, want) {
		t.Fatalf("additions = %q, want %q and nothing past the limit", got, want)
	}
}

func TestReadingThatRunsOutOfTimeKeepsWhatItFound(t *testing.T) {
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	put(t, archive, "spindle.json", []byte(sampleManifest), zip.Deflate)
	put(t, archive, "dist/backend.js", []byte(`spindle.registerTool({ name: "read_in_time" })`), zip.Deflate)
	put(t, archive, "dist/frontend.js", []byte(padding(512<<10)+`ctx.ui.registerDrawerTab({ title: "Too late" })`), zip.Store)
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	store := &stallingStore{data: file.Bytes()}
	inspected, err := probe.Inspect(context.Background(), store, uuid.New(), int64(file.Len()), "extension.zip")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	claim, ok := Spindle{}.Claim(inspected)
	if !ok {
		t.Fatal("the archive was not claimed")
	}
	store.stallFrom, store.stallTo = 128<<10, 384<<10

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	began := time.Now()
	parsed, err := Spindle{}.Parse(ctx, inspected, claim)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if elapsed := time.Since(began); elapsed > 2*time.Second {
		t.Errorf("parse took %s, want it to stop when the time is up", elapsed)
	}
	if got := additions(t, parsed); !slices.Equal(got, []string{"Tools | read_in_time"}) {
		t.Fatalf("additions = %q, want what was read in time", got)
	}
}

// padding writes about size bytes of code that registers nothing, varied the way real code is.
func padding(size int) string {
	var code strings.Builder
	for index := 0; code.Len() < size; index++ {
		fmt.Fprintf(&code, "const value%d = %d;\n", index, index*7919%104729)
	}
	return code.String()
}

// stallingStore holds back reads of one stretch of the archive until the reader gives up.
type stallingStore struct {
	data               []byte
	stallFrom, stallTo int64
}

func (store *stallingStore) ReadRange(ctx context.Context, _ uuid.UUID, offset, length int64) (io.ReadCloser, error) {
	if offset < store.stallTo && offset+length > store.stallFrom {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return io.NopCloser(bytes.NewReader(store.data[offset : offset+length])), nil
}

// additions reads what the parsed extension adds, each as its group and what it names.
func additions(t *testing.T, parsed format.Parsed) []string {
	t.Helper()
	list := elementOf(t, parsed, block.RoleExtensionAdditions).Content.(block.FieldList)
	named := make([]string, 0, len(list.Fields))
	for _, field := range list.Fields {
		if field.ID == uuid.Nil {
			t.Errorf("%q has no id", field.Value)
		}
		named = append(named, field.Name+" | "+field.Value)
	}
	return named
}
