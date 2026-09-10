import { describe, expect, test } from "bun:test";
import { posterFace, typeSetting } from "./poster-face";

const cover = { url: "/media/1/grid/1", width: 512, height: 640 };

describe("which face a catalog poster wears", () => {
  test("shows the creator's picture when there is one", () => {
    expect(posterFace({ cover })).toBe("art");
  });

  test("sets the name in type when the creator gave no picture", () => {
    expect(posterFace({ cover: null })).toBe("type");
  });

  test("sets the name in type when the picture failed to load", () => {
    expect(posterFace({ cover, failed: true })).toBe("type");
  });
});

describe("how large a type poster sets its name", () => {
  test("gives a short name the whole plate", () => {
    expect(typeSetting("Moth")).toBe("grand");
  });

  test("steps down as the name grows", () => {
    const steps = [
      typeSetting("Lantern Keeper"),
      typeSetting("A field guide to impossible weather"),
      typeSetting(
        "A field guide to impossible weather, its causes and its remedies",
      ),
      typeSetting(
        "A field guide to impossible weather, its causes and its remedies, and what to pack for each of them",
      ),
    ];

    expect(steps).toEqual(["grand", "large", "medium", "small"]);
  });

  test("treats an unnamed asset as a short name rather than an empty plate", () => {
    expect(typeSetting("")).toBe("grand");
  });

  test("measures the longest word too, so one long word still fits", () => {
    expect(typeSetting("Supercalifragilisticexpialidocious")).toBe("small");
  });

  test("counts a hyphenated name by its parts, because a line can break there", () => {
    expect(typeSetting("lumiverse-short-description")).toBe("large");
  });
});
