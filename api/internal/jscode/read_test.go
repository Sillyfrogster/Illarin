package jscode

import (
	"slices"
	"testing"
)

func TestReadFindsACallToANamedMethodWithItsLiteralArgument(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`spindle.registerTool({ name: "search_notes", description: "Searches notes" })`), "registerTool")

	if len(file.Calls) != 1 {
		t.Fatalf("calls = %+v, want one", file.Calls)
	}
	call := file.Calls[0]
	if !slices.Equal(call.Callee, []string{"spindle", "registerTool"}) {
		t.Errorf("callee = %v, want spindle.registerTool", call.Callee)
	}
	if name, ok := call.Arg(0).Property("name").Literal(); !ok || name != "search_notes" {
		t.Errorf("name = %q %t, want search_notes", name, ok)
	}
}

func TestReadPassesOverCommentsAndStringsThatOnlyMentionACall(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		// spindle.registerTool({ name: "in_a_line_comment" })
		/* spindle.registerTool({ name: "in_a_block_comment" }) */
		const note = "spindle.registerTool({ name: 'in_a_string' })"
		spindle.registerTool(/* the one that runs */ { name: "real" })
	`), "registerTool")

	if names := literalNames(file); !slices.Equal(names, []string{"real"}) {
		t.Fatalf("names = %v, want only the call that runs", names)
	}
}

func TestReadTakesATemplateAsLiteralOnlyWhenItSubstitutesNothing(t *testing.T) {
	t.Parallel()
	file := Read([]byte("spindle.registerTool({ name: `plain` })\n"+
		"spindle.registerTool({ name: `tool_${suffix}` })\n"+
		"const label = `${spindle.registerTool({ name: \"inside\" })} it's // not a comment`\n"+
		"spindle.registerTool({ name: \"after\" })\n"), "registerTool")

	if names := literalNames(file); !slices.Equal(names, []string{"plain", "inside", "after"}) {
		t.Fatalf("names = %v, want plain, inside and after", names)
	}
}

func TestReadTellsARegularExpressionFromDivision(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		const stars = /[/*]/g, quotes = /['"]/
		spindle.registerTool({ name: "after_the_patterns" })
		const ratio = total / count; spindle.registerTool({ name: "a/b" })
		const half = 1 / 2; spindle.registerTool({ name: "c/d" })
	`), "registerTool")

	want := []string{"after_the_patterns", "a/b", "c/d"}
	if names := literalNames(file); !slices.Equal(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

func TestReadListsTheModulesAFileImportsByLiteralName(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		import { alpha } from "./alpha"
		import './side-effect.js'
		export * from "./reexported"
		const lazy = await import("./lazy.js")
		const built = import("./" + name)
		const old = require('../old.cjs')
		const from = "./not-an-import"
		router.import("./not-a-module")
	`))

	want := []string{"./alpha", "./side-effect.js", "./reexported", "./lazy.js", "../old.cjs"}
	if !slices.Equal(file.Imports, want) {
		t.Fatalf("imports = %v, want %v", file.Imports, want)
	}
}

func TestReadLeavesOutModulesImportedOnlyForTheirTypes(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		import type { Beta, Gamma as G } from './types'
		import type Settings from "./settings"
		import type * as Api from "./api"
		export type { Shape } from "./shape"
		export type * from "./kinds"
		import type from "./a-default-named-type"
		import type, { delta } from "./delta"
		import { type Tool, register } from "./tools"
		export type Alias = string
		export { epsilon } from "./epsilon"
		export type { Local }
		import "./after-a-local-type-export"
	`))

	want := []string{"./a-default-named-type", "./delta", "./tools", "./epsilon", "./after-a-local-type-export"}
	if !slices.Equal(file.Imports, want) {
		t.Fatalf("imports = %v, want %v", file.Imports, want)
	}
}

func TestReadFollowsAValueIntoArraysMemberNamesAndNestedCalls(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		spindle.commands.register([
			{ id: "summarize-chat", label: "Summarize Chat" },
			{ id: "export-notes", label: labels.exportNotes },
			...extraCommands,
		])
		eventSource.on(event_types.MESSAGE_RECEIVED, onMessage)
		SlashCommandParser.addCommandObject(SlashCommand.fromProps({ name: "roll", callback: roll }))
	`), "register", "on", "addCommandObject")
	if len(file.Calls) != 3 {
		t.Fatalf("calls = %+v, want three", file.Calls)
	}

	commands, event, command := file.Calls[0], file.Calls[1], file.Calls[2]
	if !slices.Equal(commands.Callee, []string{"spindle", "commands", "register"}) {
		t.Errorf("callee = %v, want spindle.commands.register", commands.Callee)
	}
	labels := []string{}
	for _, item := range commands.Arg(0).Items() {
		if label, ok := item.Property("label").Literal(); ok {
			labels = append(labels, label)
		}
	}
	if len(commands.Arg(0).Items()) != 3 || !slices.Equal(labels, []string{"Summarize Chat"}) {
		t.Errorf("labels = %v from %d items, want the one literal label from three", labels, len(commands.Arg(0).Items()))
	}
	if member := event.Arg(0).Member(); !slices.Equal(member, []string{"event_types", "MESSAGE_RECEIVED"}) {
		t.Errorf("event = %v, want event_types.MESSAGE_RECEIVED", member)
	}
	built := command.Arg(0).Call()
	if !slices.Equal(built.Callee, []string{"SlashCommand", "fromProps"}) {
		t.Fatalf("built with %v, want SlashCommand.fromProps", built.Callee)
	}
	if name, ok := built.Arg(0).Property("name").Literal(); !ok || name != "roll" {
		t.Errorf("command name = %q %t, want roll", name, ok)
	}
	missing := event.Arg(2)
	if _, ok := missing.Literal(); ok || missing.Items() != nil || missing.Member() != nil || missing.Call().Callee != nil {
		t.Error("a missing argument read as something")
	}
}

func TestReadSpellsOutTheEscapesInALiteral(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		spindle.registerTool({ name: 'it\'s' })
		spindle.registerTool({ name: "tab\tand\nline" })
		spindle.registerTool({ name: "A\u{1F600}\x42" })
		spindle.registerTool({ name: "joined \
		line" })
	`), "registerTool")

	want := []string{"it's", "tab\tand\nline", "A😀B", "joined \t\tline"}
	if names := literalNames(file); !slices.Equal(names, want) {
		t.Fatalf("names = %q, want %q", names, want)
	}
}

func TestReadTakesAnObjectLiteralThatTypeScriptCastsToAType(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		spindle.registerTool({ name: "cast" } as ToolDefinition)
		spindle.registerTool({ name: "checked" } satisfies ToolDefinition)
		spindle.registerTool({ name: "indexed" } as Parameters<Host['ui']['registerTool']>[0])
		spindle.registerTool(({ name: "wrapped" }))
	`), "registerTool")

	if names := literalNames(file); !slices.Equal(names, []string{"cast", "checked", "indexed"}) {
		t.Fatalf("names = %v, want the three cast objects", names)
	}
}

func TestReadFindsCallsMadeThroughOptionalChainingButNotFunctionsBeingDeclared(t *testing.T) {
	t.Parallel()
	file := Read([]byte(`
		function registerTool(options) { return options }
		spindle?.registerTool({ name: "optional_owner" })
		spindle.registerTool?.({ name: "optional_call" })
		spindle.registerTool(...[{ name: "spread" }])
	`), "registerTool")

	callees := [][]string{}
	for _, call := range file.Calls {
		callees = append(callees, call.Callee)
	}
	if len(file.Calls) != 3 {
		t.Fatalf("callees = %v, want the three calls and not the declaration", callees)
	}
	for _, call := range file.Calls {
		if !slices.Equal(call.Callee, []string{"spindle", "registerTool"}) {
			t.Errorf("callee = %v, want spindle.registerTool", call.Callee)
		}
	}
	if names := literalNames(file); !slices.Equal(names, []string{"optional_owner", "optional_call"}) {
		t.Errorf("names = %v, want the two written out as objects", names)
	}
}

func literalNames(file File) []string {
	names := []string{}
	for _, call := range file.Calls {
		if name, ok := call.Arg(0).Property("name").Literal(); ok {
			names = append(names, name)
		}
	}
	return names
}
