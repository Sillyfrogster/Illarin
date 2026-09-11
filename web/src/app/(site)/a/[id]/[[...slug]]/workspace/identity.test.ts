import { describe, expect, test } from "bun:test";
import {
  BLURB_LIMIT,
  blurbCharacterCount,
  blurbLimitMessage,
} from "./identity";

describe("workspace blurb limit", () => {
  test("accepts 400 Unicode characters", () => {
    const blurb = "界".repeat(BLURB_LIMIT);

    expect(blurbCharacterCount(blurb)).toBe(400);
    expect(blurbLimitMessage(blurb)).toBe("");
  });

  test("marks all 401 submitted characters without cutting the text", () => {
    const blurb = "界".repeat(BLURB_LIMIT + 1);

    expect(blurbCharacterCount(blurb)).toBe(401);
    expect(blurbLimitMessage(blurb)).toBe(
      "The blurb must be 400 characters or fewer.",
    );
    expect(blurb).toHaveLength(401);
  });
});
