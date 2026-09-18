import { expect, test } from "bun:test";
import { previewConflicts } from "./work-publication";

test("a persisted replacement preview can omit its conflict list", () => {
  expect(previewConflicts({ conflicts: null })).toEqual([]);
  expect(previewConflicts({ conflicts: ["prompt_fragments"] })).toEqual([
    "prompt_fragments",
  ]);
});
