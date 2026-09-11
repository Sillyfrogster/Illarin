import { shortMoment } from "@/lib/dates";

export type DeliveryState =
  | "waiting"
  | "arrived"
  | "unconfirmed"
  | "gaveUp"
  | "stopped";

/** Sending is the same for a blog post and an asset update, so both read alike. */
export type Sending = {
  state: string;
  attempts: number;
  dueAt: string;
  occurredAt: string;
  settledReason?: string;
  settledAt?: string | null;
  last?: { detail: string } | undefined;
};

const OUR_DOING = new Set([
  "gone",
  "removed",
  "disabled",
  "moved",
  "withheld",
  "withdrawn",
  "unlisted",
  "deleted",
]);

export function deliveryState(one: Sending): DeliveryState {
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
  withheld: "Cancelled: this asset is withheld.",
  withdrawn: "Cancelled: this update was withdrawn.",
  unlisted:
    "Cancelled: this asset is unlisted and the update was not cleared to send its link.",
  deleted: "Cancelled: this asset is no longer published.",
};

export function deliveryStanding(one: Sending, now = new Date()): string {
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
