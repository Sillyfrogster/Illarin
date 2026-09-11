import type {
  PostDelivery,
  PublicationDestinationChoice,
  PublicationEvent,
} from "@/lib/api/query";
import { shortMoment } from "@/lib/dates";

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

export type DeliveryState =
  | "waiting"
  | "arrived"
  | "unconfirmed"
  | "gaveUp"
  | "stopped";

const OUR_DOING = new Set(["gone", "removed", "disabled", "moved"]);

export function deliveryState(one: PostDelivery): DeliveryState {
  if (one.state === "delivered") return "arrived";
  if (one.state === "unconfirmed") return "unconfirmed";
  if (one.state !== "failed") return "waiting";
  return OUR_DOING.has(one.settledReason ?? "") ? "stopped" : "gaveUp";
}

const STOPPED_WORDS: Record<string, string> = {
  gone: "The endpoint returned 410 Gone. Delivery to it has stopped.",
  removed: "The destination was removed.",
  disabled: "The destination was switched off.",
  moved: "The destination moved to another address.",
};

export function deliveryStanding(one: PostDelivery, now = new Date()): string {
  if (one.state === "delivered") {
    return `Delivered ${shortMoment(one.settledAt ?? one.occurredAt)}`;
  }
  if (one.state === "unconfirmed") {
    return "Discord accepted the request without confirming a message. Check the channel before retrying.";
  }
  if (one.state === "failed") {
    const stopped = STOPPED_WORDS[one.settledReason ?? ""];
    if (stopped) return stopped;
    if (one.settledReason === "exhausted") {
      return `Gave up after ${tries(one.attempts)}`;
    }
    return `Delivery rejected. ${one.last?.detail ?? "The destination rejected the announcement."}`;
  }
  if (one.attempts === 0) return "Queued";
  const waited = new Date(one.dueAt).getTime() - now.getTime();
  if (waited <= 0) return "Sending";
  return `Attempt ${one.attempts + 1} ${inWords(waited)}, after ${tries(one.attempts)}`;
}

const minute = 60 * 1000;
const hour = 60 * minute;

function inWords(waited: number): string {
  if (waited < minute) return "shortly";
  if (waited < hour)
    return `in ${count(Math.round(waited / minute), "minute")}`;
  return `in ${count(Math.round(waited / hour), "hour")}`;
}

function tries(made: number): string {
  return count(made, "try", "tries");
}

function count(many: number, one: string, more = `${one}s`): string {
  return `${many} ${many === 1 ? one : more}`;
}
