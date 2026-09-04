import type { PostAction, PostRevision } from "@/lib/api/query";

/** One thing the editorial history shows, in the order it happened. */
export type HistoryEntry =
  | { kind: "edition"; at: string; revision: PostRevision }
  | { kind: "note"; at: string; action: PostAction };

// keptBy names the actions an edition card already accounts for.
const keptBy = new Set(["post.checkpointed", "post.published"]);

/** The editions and what was done around them as one stream, newest first. */
export function historyStream(
  revisions: PostRevision[],
  actions: PostAction[],
): HistoryEntry[] {
  const editions: HistoryEntry[] = revisions.map((revision) => ({
    kind: "edition",
    at: revision.capturedAt,
    revision,
  }));
  const notes: HistoryEntry[] = actions
    .filter((action) => !keptBy.has(action.action))
    .map((action) => ({ kind: "note", at: action.at, action }));
  return [...editions, ...notes].sort(
    (one, other) => Date.parse(other.at) - Date.parse(one.at),
  );
}

/** What one recorded action reads as after the name of who did it. */
export function noteWords(done: PostAction): string {
  switch (done.action) {
    case "post.created":
      return "started this post";
    case "post.revision.restored":
      return done.revision
        ? `restored edition ${done.revision}`
        : "restored an edition";
    case "post.address.corrected":
      return "corrected the address";
    case "post.byline.corrected":
      return "corrected the name on the post";
    case "post.checkpointed":
      return "kept an edition";
    case "post.published":
      return "published an edition";
    default:
      return done.action;
  }
}

/** What one kept edition reads as beside its number. */
export function revisionWords(capturedFor: string): string {
  return capturedFor === "publication" ? "Published" : "Kept while writing";
}
