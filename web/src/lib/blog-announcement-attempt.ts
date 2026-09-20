import type {
  BlogAnnouncementType,
  BlogIntegrationChoice,
} from "@/lib/api/query";

export const ANNOUNCEMENT_WORDS: Record<
  BlogAnnouncementType,
  { word: string; what: string }
> = {
  "blog.post.published.v1": {
    word: "Published",
    what: "A post is published or republished.",
  },
  "blog.post.updated.v1": {
    word: "Updated",
    what: "Changes to a published post are published.",
  },
  "blog.post.unpublished.v1": {
    word: "Unpublished",
    what: "A post is unpublished from public view.",
  },
};

export const EVENTS: BlogAnnouncementType[] = [
  "blog.post.published.v1",
  "blog.post.updated.v1",
  "blog.post.unpublished.v1",
];

export function announcementWord(type: string): string {
  const held = EVENTS.find((one) => one === type);
  return held ? ANNOUNCEMENT_WORDS[held].word : "Publication";
}

export type Transition = "publish" | "changes" | "unpublish" | "republish";

export function transitionEvent(transition: Transition): BlogAnnouncementType {
  if (transition === "changes") return "blog.post.updated.v1";
  if (transition === "unpublish") return "blog.post.unpublished.v1";
  return "blog.post.published.v1";
}

export function offeredFor(
  integration: BlogIntegrationChoice,
  event: BlogAnnouncementType,
  announced: boolean,
): boolean {
  if (!integration.announcements.includes(event)) return false;
  return integration.type !== "discord" || !announced;
}
