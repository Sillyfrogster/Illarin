import type { RecentVersion } from "@/lib/api/shapes";

export const FEATURED_LIMIT = 4;

/** Pins a work to the end of the featured four, or unpins it */
export function toggleFeatured(featured: string[], workId: string): string[] {
  if (featured.includes(workId)) {
    return featured.filter((id) => id !== workId);
  }
  if (featured.length >= FEATURED_LIMIT) return featured;
  return [...featured, workId];
}

export function countWords(
  works: number,
  followers: number,
  noun = "work",
): string {
  return `${plural(works, noun)} · ${plural(followers, "follower")}`;
}

function plural(count: number, noun: string): string {
  return `${count.toLocaleString("en-GB")} ${noun}${count === 1 ? "" : "s"}`;
}

/** Says which version it was and when */
export function versionWords(version: RecentVersion, now = new Date()): string {
  const number = version.initial
    ? "Published"
    : `v${version.number}${version.versionLabel ? ` · ${version.versionLabel}` : ""}`;
  return `${number} · ${shortDate(version.recordedAt, now)}`;
}

function shortDate(value: string, now: Date): string {
  const at = new Date(value);
  return at.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: at.getFullYear() === now.getFullYear() ? undefined : "numeric",
  });
}

export type FoldedVersion = { version: RecentVersion; earlier: number };

/** Folds a run of back-to-back versions of one work into its newest, so one busy work does not fill the list. */
export function foldVersions(versions: RecentVersion[]): FoldedVersion[] {
  const folded: FoldedVersion[] = [];
  for (const version of versions) {
    const last = folded.at(-1);
    if (last && last.version.workId === version.workId) {
      folded[folded.length - 1] = { ...last, earlier: last.earlier + 1 };
    } else {
      folded.push({ version, earlier: 0 });
    }
  }
  return folded;
}
