import type { Post, PostMedia, PublicationApp } from "@/lib/api/query";
import type { PostDocument } from "@/lib/post-document";
import { asPostDocument } from "@/lib/post-document";
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
  document: PostDocument;
  release: { appId: string; version: string; address: string } | null;
  header: { mediaId: string; alt: string; caption: string } | null;
  socialMediaId: string | null;
};

export type PublicationAction =
  | "publish"
  | "schedule"
  | "withdraw"
  | "republish"
  | "delete"
  | "recover";

export type PostNotice = {
  kind: "deleted" | "withdrawn" | "scheduled";
  tone: "accent" | "stop";
  heading: string;
  said: string;
  meanwhile: string;
  record?: string;
};

const STANDING_NAMES = {
  draft: "a draft",
  published: "published",
  withdrawn: "withdrawn",
} as const;

const STANDING: Record<Lifecycle, string> = {
  draft: "Saved privately",
  published: "Published",
  withdrawn: "Withdrawn",
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

export function publicationLabel(post: Post): string {
  const standing = writerStanding(post);
  if (standing === "deleted") return "Restore post";
  if (standing === "withdrawn") return "Republish post";
  return standing === "published" ? "Publish changes" : "Publish";
}

export function publicationActions(
  post: Post,
  admin: boolean,
): PublicationAction[] {
  const standing = writerStanding(post);
  if (standing === "deleted") return ["recover"];
  if (standing === "withdrawn") {
    return mayDelete(post, admin) ? ["republish", "delete"] : ["republish"];
  }
  if (standing === "published") return ["publish", "schedule", "withdraw"];
  return mayDelete(post, admin)
    ? ["publish", "schedule", "delete"]
    : ["publish", "schedule"];
}

function mayDelete(post: Post, admin: boolean): boolean {
  if (post.status === "published") return false;
  return admin || post.publishedAt === undefined;
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
  const withdrawal = post.withdrawal;
  if (post.status === "withdrawn" && withdrawal) {
    notices.push({
      kind: "withdrawn",
      tone: "stop",
      heading: "Withdrawn",
      said: `Withdrawn by @${withdrawal.by}.`,
      meanwhile:
        withdrawal.explanation ||
        "Readers see the default withdrawal message. No public explanation was provided.",
      record: withdrawal.reason,
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
): "draft" | "published" | "withdrawn" {
  if (post.status === "published") return "published";
  return post.status === "withdrawn" ? "withdrawn" : "draft";
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
    document: asPostDocument(post.document),
    header: post.header
      ? {
          mediaId: post.header.mediaId,
          alt: post.header.alt,
          caption: post.header.caption ?? "",
        }
      : null,
    socialMediaId: post.socialMediaId ?? null,
    release: post.release
      ? {
          appId: post.release.app.id,
          version: post.release.version,
          address: post.release.address ?? "",
        }
      : null,
  };
}

export function mergedMedia(
  held: PostMedia[],
  saved: PostMedia[],
): PostMedia[] {
  const known = new Set(saved.map((one) => one.id));
  return [...saved, ...held.filter((one) => !known.has(one.id))];
}

export function namedRelease(
  open: PublicationApp[],
  post: Post,
): PublicationApp[] {
  const named = post.release?.app;
  if (!named || open.some((app) => app.id === named.id)) return open;
  return [named, ...open];
}
