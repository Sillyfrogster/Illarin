import type {
  PublicationDestinationChoice,
  PublicationEvent,
} from "@/lib/api/query";

export const EVENT_WORDS: Record<
  PublicationEvent,
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

export const EVENTS: PublicationEvent[] = [
  "publication.post.published.v1",
  "publication.post.updated.v1",
  "publication.post.withdrawn.v1",
];

export function eventWord(type: string): string {
  const held = EVENTS.find((one) => one === type);
  return held ? EVENT_WORDS[held].word : "Publication";
}

export type Transition = "publish" | "changes" | "withdraw" | "republish";

export function transitionEvent(transition: Transition): PublicationEvent {
  if (transition === "changes") return "publication.post.updated.v1";
  if (transition === "withdraw") return "publication.post.withdrawn.v1";
  return "publication.post.published.v1";
}

export function offeredFor(
  destination: PublicationDestinationChoice,
  event: PublicationEvent,
  announced: boolean,
): boolean {
  if (!destination.events.includes(event)) return false;
  return destination.kind !== "discord" || !announced;
}
