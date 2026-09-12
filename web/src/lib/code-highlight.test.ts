import { expect, test } from "bun:test";
import { highlightCode, LANGUAGE_LABELS } from "./code-highlight";
import { POST_LANGUAGES } from "./post-document";

test("every language Illarin labels code with can be highlighted and named", () => {
  for (const name of POST_LANGUAGES) {
    expect(LANGUAGE_LABELS[name]).toBeTruthy();
    const runs = highlightCode("one = 1", name);
    expect(runs.map((run) => run.text).join("")).toBe("one = 1");
  }
});

test("a run keeps the part it plays in the code", () => {
  const runs = highlightCode(
    'func main() {\n\t// note\n\tprint("hi")\n}',
    "go",
  );
  const roles = new Map(runs.map((run) => [run.text, run.role]));
  expect(roles.get("func")).toBe("keyword");
  expect(roles.get("// note")).toBe("comment");
  expect(roles.get('"hi"')).toBe("literal");
});

test("plain text is one unhighlighted run", () => {
  expect(highlightCode("illarin publish --dry-run", "plain")).toEqual([
    { text: "illarin publish --dry-run" },
  ]);
});

test("highlighting never loses or reorders a character", () => {
  const source = "select id\n  from posts -- every one\n where slug = 'a';";
  expect(
    highlightCode(source, "sql")
      .map((run) => run.text)
      .join(""),
  ).toBe(source);
});
