import type { PostAction, PostRevision } from "@/lib/api/query";

export type HistoryEntry =
  | { kind: "edition"; at: string; revision: PostRevision }
  | { kind: "note"; at: string; action: PostAction };

const keptBy = new Set([
  "post.checkpointed",
  "post.published",
  "post.scheduled",
]);

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
    case "post.schedule.replaced":
      return done.revision
        ? `set edition ${done.revision} to go live instead`
        : "changed what goes live";
    case "post.schedule.cancelled":
      return "stopped this going live";
    case "post.schedule.stopped":
      return "could not publish the edition that was waiting";
    default:
      return done.action;
  }
}

export function revisionWords(capturedFor: string): string {
  if (capturedFor === "publication") return "Published";
  if (capturedFor === "schedule") return "Kept to go live later";
  return "Kept while writing";
}
