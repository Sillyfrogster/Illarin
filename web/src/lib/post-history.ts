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
        ? `restored revision ${done.revision}`
        : "restored a revision";
    case "post.address.corrected":
      return "corrected the address";
    case "post.byline.corrected":
      return "corrected the name on the post";
    case "post.checkpointed":
      return "saved a checkpoint";
    case "post.published":
      return "published a revision";
    case "post.schedule.replaced":
      return done.revision
        ? `scheduled revision ${done.revision} instead`
        : "changed the scheduled revision";
    case "post.schedule.cancelled":
      return "cancelled scheduled publication";
    case "post.schedule.stopped":
      return "could not publish the scheduled revision";
    default:
      return done.action;
  }
}

export function revisionWords(capturedFor: string): string {
  if (capturedFor === "publish") return "Published";
  if (capturedFor === "schedule") return "Scheduled";
  return "Checkpoint";
}
