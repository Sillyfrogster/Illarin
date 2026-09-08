import type { RecordedVersion } from "@/lib/api/query";

/** What one recorded version is called, which never presents a version captured from what an asset already was as an update somebody published. */
export function versionTitle(version: RecordedVersion): string {
  if (version.initial) return "Initial recording";
  if (version.number === 1) return "First published";
  return `Update ${version.number}`;
}

/** What a recorded version says it holds, where its creator wrote nothing. */
export function versionSummary(version: RecordedVersion, kind: string): string {
  if (version.summary) return version.summary;
  if (version.initial) {
    return `Recorded from what this ${kind} already was when Illarin began keeping history.`;
  }
  return `The version this ${kind} was first published as.`;
}

/** The day a version was recorded, in the words the rest of the site uses. */
export function versionDate(version: RecordedVersion): string {
  return new Date(version.recordedAt).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
