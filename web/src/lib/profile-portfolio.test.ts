import { expect, test } from "bun:test";
import type { RecentVersion } from "@/lib/api/shapes";
import {
  countWords,
  foldVersions,
  toggleFeatured,
  versionWords,
} from "./profile-portfolio";

test("pinning adds to the end, unpinning removes, and a full row refuses a fifth", () => {
  expect(toggleFeatured([], "a")).toEqual(["a"]);
  expect(toggleFeatured(["a", "b"], "a")).toEqual(["b"]);
  expect(toggleFeatured(["a", "b", "c", "d"], "e")).toEqual([
    "a",
    "b",
    "c",
    "d",
  ]);
  expect(toggleFeatured(["a", "b", "c", "d"], "b")).toEqual(["a", "c", "d"]);
});

test("counts read as words with the right plural", () => {
  expect(countWords(1, 0)).toBe("1 work · 0 followers");
  expect(countWords(108, 842)).toBe("108 works · 842 followers");
  expect(countWords(2, 1)).toBe("2 works · 1 follower");
});

test("a version line says which version and when, and a first publication says Published", () => {
  const now = new Date("2026-09-21T12:00:00Z");
  const version: RecentVersion = {
    cover: null,
    initial: false,
    number: 4,
    recordedAt: "2026-09-20T10:00:00Z",
    summary: "Added two greetings.",
    versionLabel: "",
    workId: "w",
    workName: "The last archivist",
    workType: "character",
  };
  expect(versionWords(version, now)).toBe("v4 · 20 Sept");
  expect(versionWords({ ...version, versionLabel: "2.1" }, now)).toBe(
    "v4 · 2.1 · 20 Sept",
  );
  expect(
    versionWords(
      {
        ...version,
        initial: true,
        number: 1,
        recordedAt: "2025-03-02T10:00:00Z",
      },
      now,
    ),
  ).toBe("Published · 2 Mar 2025");
});

test("a run of versions of one work folds into its newest, counting the rest", () => {
  const at = (workId: string, number: number): RecentVersion => ({
    cover: null,
    initial: false,
    number,
    recordedAt: "2026-09-20T10:00:00Z",
    summary: "",
    versionLabel: "",
    workId,
    workName: workId,
    workType: "character",
  });
  const folded = foldVersions([
    at("a", 5),
    at("a", 4),
    at("a", 3),
    at("b", 2),
    at("a", 2),
  ]);
  expect(
    folded.map(({ version, earlier }) => [
      version.workId,
      version.number,
      earlier,
    ]),
  ).toEqual([
    ["a", 5, 2],
    ["b", 2, 0],
    ["a", 2, 0],
  ]);
});
