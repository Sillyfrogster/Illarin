import { expect, test } from "bun:test";
import { PORTRAIT_GROUNDS, portraitGround } from "./portrait-tone";

test("gives a handle the same ground every time", () => {
  expect(portraitGround("someone")).toBe(portraitGround("someone"));
});

test("spreads handles across every ground there is", () => {
  const handles = Array.from({ length: 200 }, (_, at) => `creator${at}`);
  const used = new Set(handles.map(portraitGround));

  expect(used.size).toBe(PORTRAIT_GROUNDS.length);
});

test("only ever answers with a ground the page knows how to paint", () => {
  for (const handle of ["a", "zzzzzz", "a.long.handle_with_marks", "9"]) {
    expect(PORTRAIT_GROUNDS).toContain(portraitGround(handle));
  }
});

test("a handle with no characters still gets a ground", () => {
  expect(PORTRAIT_GROUNDS).toContain(portraitGround(""));
});
