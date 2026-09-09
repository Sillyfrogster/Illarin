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

// How much of a note an entry shows before it folds the rest away, about four lines of it.
const NOTE_FOLD = 280;

/** Whether a creator's note runs long enough that an entry folds it. */
export function isLongNote(notes: string): boolean {
  return notes.length > NOTE_FOLD;
}

/** Everything Illarin recorded before this version, newest first. */
export function earlierVersions(
  versions: RecordedVersion[],
  version: RecordedVersion,
): RecordedVersion[] {
  return versions
    .filter((one) => one.number < version.number)
    .sort((one, other) => other.number - one.number);
}

/** The address a version answers to, which the rail and the entry both use. */
export function versionAnchor(version: RecordedVersion): string {
  return `version-${version.number}`;
}
