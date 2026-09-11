import { describe, expect, test } from "bun:test";
import {
  BLURB_LIMIT,
  blurbCharacterCount,
  blurbLimitMessage,
  identityHasChanged,
} from "./identity";

describe("workspace blurb limit", () => {
  test("accepts 400 Unicode characters", () => {
    const blurb = "🌙".repeat(BLURB_LIMIT);

    expect(blurbCharacterCount(blurb)).toBe(400);
    expect(blurbLimitMessage(blurb)).toBe("");
  });

  test("marks all 401 submitted characters without cutting the text", () => {
    const blurb = "🌙".repeat(BLURB_LIMIT + 1);

    expect(blurbCharacterCount(blurb)).toBe(401);
    expect(blurbLimitMessage(blurb)).toBe(
      "The blurb must be 400 characters or fewer.",
    );
    expect(blurb).toBe("🌙".repeat(401));
  });
});

describe("workspace identity changes", () => {
  const saved = { blurb: "", isNsfw: false, name: "Fixture asset" };

  test.each([
    ["adds", "A new pitch."],
    ["edits", "A clearer pitch."],
    ["clears", ""],
  ])("marks the workspace unsaved when a creator %s a blurb", (_, blurb) => {
    const before = blurb === "" ? { ...saved, blurb: "Old pitch." } : saved;

    expect(identityHasChanged({ ...saved, blurb }, before)).toBe(true);
  });

  test("leaves the workspace saved when the blurb matches", () => {
    expect(identityHasChanged(saved, { ...saved })).toBe(false);
  });
});
