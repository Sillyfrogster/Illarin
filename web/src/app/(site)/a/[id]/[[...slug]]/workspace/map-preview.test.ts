import { describe, expect, test } from "bun:test";
import type { WorkElement } from "@/lib/api/query";
import { previewElement } from "./map-preview";

describe("previewElement", () => {
  test("keeps the first few items and the start of long text, and nothing else changes", () => {
    const entries = Array.from({ length: 120 }, (_, at) => ({
      content: "x".repeat(5000),
      name: `entry ${at}`,
    }));
    const element = {
      content: { entries },
      id: "lore",
      type: "entry_table",
    } as unknown as WorkElement;

    const preview = previewElement(element);
    const shown = (preview.content as unknown as { entries: typeof entries })
      .entries;

    expect(shown).toHaveLength(4);
    expect(shown[0].name).toBe("entry 0");
    expect(shown[0].content.length).toBeLessThan(5000);
    expect(preview.id).toBe("lore");
    expect(previewElement(element)).toBe(preview);
  });
});
