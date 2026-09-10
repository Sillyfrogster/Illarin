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
    word: "Publication",
    what: "A post reaching the blog, and one put back after a takedown.",
  },
  "publication.post.updated.v1": {
    word: "Changes",
    what: "A post already on the blog, published again.",
  },
  "publication.post.withdrawn.v1": {
    word: "Takedown",
    what: "A post leaving the blog. Its address answers 410 from then on.",
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
  gone: "It answered 410, so nothing goes there again.",
  removed: "The destination was removed.",
  disabled: "The destination was switched off.",
  moved: "The destination moved to another address.",
};

export function deliveryStanding(one: PostDelivery, now = new Date()): string {
  if (one.state === "delivered") {
    return `Arrived ${shortMoment(one.settledAt ?? one.occurredAt)}`;
  }
  if (one.state === "unconfirmed") {
    return "Discord took it but never said which message it made. Look in the channel to see whether it arrived.";
  }
  if (one.state === "failed") {
    const stopped = STOPPED_WORDS[one.settledReason ?? ""];
    if (stopped) return stopped;
    if (one.settledReason === "exhausted") {
      return `Gave up after ${tries(one.attempts)}`;
    }
    return `Turned away. ${one.last?.detail ?? "It would not take it."}`;
  }
  if (one.attempts === 0) return "Waiting to go out";
  const waited = new Date(one.dueAt).getTime() - now.getTime();
  if (waited <= 0) return "Going out now";
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
