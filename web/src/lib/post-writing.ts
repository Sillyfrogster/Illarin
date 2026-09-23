import type { Post, PostMedia } from "@/lib/api/query";
import type { PostBody } from "@/lib/post-body";
import { asPostBody } from "@/lib/post-body";
import { type Lifecycle, lifecycleOf } from "@/lib/post-standing";

export type Saving =
  | "clean"
  | "dirty"
  | "saving"
  | "saved"
  | "conflict"
  | "refused";

export type Draft = {
  categoryId: string;
  title: string;
  summary: string;
  slug: string;
  body: PostBody;
  header: { mediaId: string; alt: string; caption: string } | null;
  linkCardMediaId: string | null;
};

export type PublishAction =
  | "publish"
  | "schedule"
  | "unpublish"
  | "republish"
  | "recover";

export type PostNotice = {
  kind: "deleted" | "unpublished" | "scheduled";
  tone: "accent" | "stop";
  heading: string;
  said: string;
  meanwhile: string;
  record?: string;
};

const STANDING_NAMES = {
  draft: "a draft",
  published: "published",
  unpublished: "unpublished",
} as const;

const STANDING: Record<Lifecycle, string> = {
  draft: "Saved privately",
  published: "Published",
  unpublished: "Unpublished",
  deleted: "Deleted",
};

export function savingWords(state: Saving, post: Post): string {
  switch (state) {
    case "saving":
      return "Saving";
    case "saved":
      return "Saved";
    case "dirty":
      return "Unsaved";
    case "conflict":
      return "Someone else saved this post";
    case "refused":
      return "Not saved";
    default:
      return STANDING[writerStanding(post)];
  }
}

export function writerStanding(post: Post): Lifecycle {
  return lifecycleOf(post);
}

export function publishLabel(post: Post): string {
  const standing = writerStanding(post);
  if (standing === "deleted") return "Restore post";
  if (standing === "unpublished") return "Republish post";
  return standing === "published" ? "Publish changes" : "Publish";
}

export function publishActions(post: Post): PublishAction[] {
  const standing = writerStanding(post);
  if (standing === "deleted") return ["recover"];
  if (standing === "unpublished") return ["republish"];
  if (standing === "published") return ["publish", "schedule", "unpublish"];
  return ["publish", "schedule"];
}

export function postNotices(post: Post): PostNotice[] {
  const deletion = post.deletion;
  if (deletion) {
    return [
      {
        kind: "deleted",
        tone: "stop",
        heading: "Deleted",
        said: `Deleted by @${deletion.by}. Restore the post to continue editing.`,
        meanwhile:
          "After the recovery deadline, its content, revisions and images are permanently deleted.",
        record: `Deleted while ${STANDING_NAMES[standingBeforeDeletion(post)]}.`,
      },
    ];
  }

  const notices: PostNotice[] = [];
  const unpublishing = post.unpublishing;
  if (post.status === "unpublished" && unpublishing) {
    notices.push({
      kind: "unpublished",
      tone: "stop",
      heading: "Unpublished",
      said: `Unpublished by @${unpublishing.by}.`,
      meanwhile:
        unpublishing.explanation ||
        "Readers see the default unpublishing message. No public explanation was provided.",
      record: unpublishing.reason,
    });
  }

  const schedule = post.schedule;
  const waiting =
    schedule &&
    (schedule.state === "pending" ||
      schedule.state === "publishing" ||
      schedule.state === "stopped");
  if (schedule && waiting) {
    notices.push({
      kind: "scheduled",
      tone: schedule.state === "stopped" ? "stop" : "accent",
      heading: scheduleHeading(schedule.state),
      said:
        schedule.state === "stopped"
          ? schedule.stoppedBecause || "Scheduled publication failed."
          : `Revision ${schedule.revisionNumber} ${schedule.state === "publishing" ? "is going live now" : "goes live"}`,
      meanwhile:
        schedule.state === "stopped"
          ? "Nothing was published. Schedule it again when you are ready."
          : post.status === "published"
            ? "Readers keep the currently published revision until then."
            : "Readers cannot see this post until then.",
    });
  }
  return notices;
}

function standingBeforeDeletion(
  post: Post,
): "draft" | "published" | "unpublished" {
  if (post.status === "published") return "published";
  return post.status === "unpublished" ? "unpublished" : "draft";
}

function scheduleHeading(state: string): string {
  if (state === "stopped") return "Scheduled publication failed";
  if (state === "publishing") return "Publishing";
  return "Scheduled";
}

export function draftFromPost(post: Post): Draft {
  return {
    categoryId: post.category.id,
    title: post.title,
    summary: post.summary,
    slug: post.slug,
    body: asPostBody(post.body),
    header: post.header
      ? {
          mediaId: post.header.mediaId,
          alt: post.header.alt,
          caption: post.header.caption ?? "",
        }
      : null,
    linkCardMediaId: post.linkCardMediaId ?? null,
  };
}

export function mergedMedia(
  held: PostMedia[],
  saved: PostMedia[],
): PostMedia[] {
  const known = new Set(saved.map((one) => one.id));
  return [...saved, ...held.filter((one) => !known.has(one.id))];
}
