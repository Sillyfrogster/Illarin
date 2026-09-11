import type { Post } from "@/lib/api/query";

export type Lifecycle = "draft" | "published" | "withdrawn" | "deleted";

export type Standing = "everything" | Lifecycle | "scheduled";

export const STANDINGS: Standing[] = [
  "everything",
  "draft",
  "scheduled",
  "published",
  "withdrawn",
  "deleted",
];

const STANDING_NAMES: Record<Standing, string> = {
  everything: "All posts",
  draft: "Drafts",
  scheduled: "Scheduled",
  published: "Published",
  withdrawn: "Withdrawn",
  deleted: "Deleted",
};

const LIFECYCLE_NAMES: Record<Lifecycle, string> = {
  draft: "Draft",
  published: "Published",
  withdrawn: "Withdrawn",
  deleted: "Deleted",
};

const NOTHING_THERE: Record<Standing, string> = {
  everything: "No posts yet. Create a post to start a private draft.",
  draft: "No draft posts.",
  scheduled: "No scheduled posts.",
  published: "No published posts.",
  withdrawn: "No withdrawn posts.",
  deleted: "No deleted posts to restore.",
};

export function standingName(standing: Standing): string {
  return STANDING_NAMES[standing];
}

export function lifecycleName(state: Lifecycle): string {
  return LIFECYCLE_NAMES[state];
}

export function nothingThere(standing: Standing): string {
  return NOTHING_THERE[standing];
}

export function lifecycleOf(post: Post): Lifecycle {
  if (post.deletion) return "deleted";
  if (post.status === "published") return "published";
  if (post.status === "withdrawn") return "withdrawn";
  return "draft";
}

export function goingLiveAt(post: Post): string | null {
  const schedule = post.schedule;
  if (!schedule) return null;
  const onItsWay =
    schedule.state === "pending" || schedule.state === "publishing";
  return onItsWay ? schedule.at : null;
}

export function inStanding(post: Post, standing: Standing): boolean {
  if (standing === "everything" || standing === "deleted") return true;
  if (standing === "scheduled") return goingLiveAt(post) !== null;
  return lifecycleOf(post) === standing;
}
