import type { RecordedVersion } from "@/lib/api/query";

export function versionTitle(version: RecordedVersion): string {
  if (version.initial) return "Initial recording";
  if (version.number === 1) return "First published";
  return `Update ${version.number}`;
}

export function versionSummary(version: RecordedVersion, kind: string): string {
  if (version.summary) return version.summary;
  if (version.initial) {
    return `Recorded from what this ${kind} already was when Illarin began keeping history.`;
  }
  return `The version this ${kind} was first published as.`;
}

export function versionDate(version: RecordedVersion): string {
  return new Date(version.recordedAt).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

const NOTE_FOLD = 280;

export function isLongNote(notes: string): boolean {
  return notes.length > NOTE_FOLD;
}

export function earlierVersions(
  versions: RecordedVersion[],
  version: RecordedVersion,
): RecordedVersion[] {
  return versions
    .filter((one) => one.number < version.number)
    .sort((one, other) => other.number - one.number);
}

export function versionAnchor(version: RecordedVersion): string {
  return `version-${version.number}`;
}
