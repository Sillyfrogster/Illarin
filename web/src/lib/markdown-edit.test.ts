import { describe, expect, test } from "bun:test";
import { applyMarkdown, type MarkdownAction } from "./markdown-edit";

function at(text: string): { body: string; start: number; end: number } {
  const start = text.indexOf("|");
  const rest = text.slice(0, start) + text.slice(start + 1);
  const end = rest.indexOf("|");
  return end === -1
    ? { body: rest, end: start, start }
    : { body: rest.slice(0, end) + rest.slice(end + 1), end, start };
}

function run(action: MarkdownAction, marked: string): string {
  const { body, start, end } = at(marked);
  const edit = applyMarkdown(action, body, { end, start });
  return (
    edit.text.slice(0, edit.selection.start) +
    "|" +
    edit.text.slice(edit.selection.start, edit.selection.end) +
    (edit.selection.end === edit.selection.start ? "" : "|") +
    edit.text.slice(edit.selection.end)
  );
}

describe("writing markdown into a field", () => {
  test("bold wraps the selection and keeps it selected", () => {
    expect(run("bold", "a |quiet| room")).toBe("a **|quiet|** room");
  });

  test("bold on an empty selection leaves the caret between the markers", () => {
    expect(run("bold", "a | room")).toBe("a **|** room");
  });

  test("bold again unwraps what it wrapped", () => {
    expect(run("bold", "a **|quiet|** room")).toBe("a |quiet| room");
  });

  test("bold unwraps when the markers are inside the selection", () => {
    expect(run("bold", "a |**quiet**| room")).toBe("a |quiet| room");
  });

  test("italic and bold do not fight over the same markers", () => {
    expect(run("italic", "a **|quiet|** room")).toBe("a **_|quiet|_** room");
  });

  test("code wraps in backticks", () => {
    expect(run("code", "set |temperature| to 1")).toBe(
      "set `|temperature|` to 1",
    );
  });

  test("a selection's surrounding spaces stay outside the markers", () => {
    expect(run("bold", "a | quiet | room")).toBe("a  **|quiet|**  room");
  });

  test("link wraps the selection and selects the address", () => {
    expect(run("link", "see the |nudges| below")).toBe(
      "see the [nudges](|url|) below",
    );
  });

  test("link with no selection writes both halves", () => {
    expect(run("link", "see |")).toBe("see [text](|url|)");
  });

  test("a bullet prefixes every line the selection touches", () => {
    expect(run("bullet", "|one\ntwo\nthree|")).toBe("|- one\n- two\n- three|");
  });

  test("a bullet again removes the prefix", () => {
    expect(run("bullet", "|- one\n- two|")).toBe("|one\ntwo|");
  });

  test("a numbered list counts from one", () => {
    expect(run("numbered", "|one\ntwo\nthree|")).toBe(
      "|1. one\n2. two\n3. three|",
    );
  });

  test("a numbered list again removes any numbering", () => {
    expect(run("numbered", "|1. one\n2. two|")).toBe("|one\ntwo|");
  });

  test("a quote prefixes the line the caret sits on", () => {
    expect(run("quote", "before\nthe l|ine\nafter")).toBe(
      "before\n> the l|ine\nafter",
    );
  });

  test("a prefix keeps a line's indentation", () => {
    expect(run("bullet", "|  one\n  two|")).toBe("|  - one\n  - two|");
  });

  test("a blank line takes no list marker", () => {
    expect(run("bullet", "|one\n\ntwo|")).toBe("|- one\n\n- two|");
  });
});
