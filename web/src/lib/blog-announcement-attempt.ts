import type {
  BlogAnnouncementType,
  BlogIntegrationChoice,
} from "@/lib/api/query";

export const ANNOUNCEMENT_WORDS: Record<
  BlogAnnouncementType,
  { word: string; what: string }
> = {
  "publication.post.published.v1": {
    word: "Published",
    what: "A post is published or republished.",
  },
  "publication.post.updated.v1": {
    word: "Updated",
    what: "Changes to a published post are published.",
  },
  "publication.post.withdrawn.v1": {
    word: "Withdrawn",
    what: "A post is withdrawn from public view.",
  },
};

export const EVENTS: BlogAnnouncementType[] = [
  "publication.post.published.v1",
  "publication.post.updated.v1",
  "publication.post.withdrawn.v1",
];

export function announcementWord(type: string): string {
  const held = EVENTS.find((one) => one === type);
  return held ? ANNOUNCEMENT_WORDS[held].word : "Publication";
}

export type Transition = "publish" | "changes" | "withdraw" | "republish";

export function transitionEvent(transition: Transition): BlogAnnouncementType {
  if (transition === "changes") return "publication.post.updated.v1";
  if (transition === "withdraw") return "publication.post.withdrawn.v1";
  return "publication.post.published.v1";
}

export function offeredFor(
  integration: BlogIntegrationChoice,
  event: BlogAnnouncementType,
  announced: boolean,
): boolean {
  if (!integration.announcements.includes(event)) return false;
  return integration.type !== "discord" || !announced;
}
