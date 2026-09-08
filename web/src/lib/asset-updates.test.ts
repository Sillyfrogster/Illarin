import { expect, test } from "bun:test";
import type { RecordedVersion } from "@/lib/api/query";
import { versionSummary, versionTitle } from "./asset-updates";

function version(over: Partial<RecordedVersion>): RecordedVersion {
  return {
    id: "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a",
    number: 1,
    recordedAt: "2026-03-12T09:00:00Z",
    initial: false,
    versionLabel: "",
    summary: "",
    notes: "",
    ...over,
  };
}

test("names a captured version a recording rather than an update", () => {
  const captured = version({ initial: true });

  expect(versionTitle(captured)).toBe("Initial recording");
  expect(versionSummary(captured, "character")).toBe(
    "Recorded from what this character already was when Illarin began keeping history.",
  );
});

test("names a first publication for what it is", () => {
  expect(versionTitle(version({}))).toBe("First published");
  expect(versionSummary(version({}), "preset")).toBe(
    "The version this preset was first published as.",
  );
});

test("names a later version by its update number and keeps the creator's summary", () => {
  const third = version({ number: 3, summary: "Rewrote the opening." });

  expect(versionTitle(third)).toBe("Update 3");
  expect(versionSummary(third, "character")).toBe("Rewrote the opening.");
});
