import { expect, test } from "bun:test";
import { codeRuns } from "./code-runs";

test("an http example marks its request line, header names and json body", () => {
  const runs = codeRuns(
    'POST /api/v1/publication/posts\nAuthorization: Bearer ip1.…\n\n{"title": "Hello"}',
    "http",
  );
  expect(runs.slice(0, 5)).toEqual([
    { text: "POST", role: "keyword" },
    { text: " /api/v1/publication/posts\n" },
    { text: "Authorization", role: "name" },
    { text: ": Bearer ip1.…" },
    { text: "\n\n" },
  ]);
  expect(
    runs.some((run) => run.text === '"title"' && run.role === "name"),
  ).toBe(true);
  expect(runs.map((run) => run.text).join("")).toBe(
    'POST /api/v1/publication/posts\nAuthorization: Bearer ip1.…\n\n{"title": "Hello"}',
  );
});

test("a post language is highlighted and anything else is left plain", () => {
  expect(codeRuns('{"a": 1}', "json").length).toBeGreaterThan(1);
  expect(codeRuns("def f():\n  pass", "python").length).toBeGreaterThan(1);
  expect(codeRuns("anything", "cobol")).toEqual([{ text: "anything" }]);
  expect(codeRuns("anything", "plain")).toEqual([{ text: "anything" }]);
});

test("every kind of example carries a label people can read", () => {
  expect(codeRuns("x", "http")).toEqual([{ text: "x" }]);
});
