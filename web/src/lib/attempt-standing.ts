import { shortMoment } from "@/lib/dates";

export type AttemptState =
  | "waiting"
  | "arrived"
  | "unconfirmed"
  | "gaveUp"
  | "stopped";

/** Sending is the same for a blog post and a work update, so both read alike. */
export type Sending = {
  state: string;
  tries: number;
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
  "unpublished",
  "unlisted",
  "deleted",
]);

export function attemptState(one: Sending): AttemptState {
  if (one.state === "delivered") return "arrived";
  if (one.state === "unconfirmed") return "unconfirmed";
  if (one.state !== "failed") return "waiting";
  return OUR_DOING.has(one.settledReason ?? "") ? "stopped" : "gaveUp";
}

const STOPPED_WORDS: Record<string, string> = {
  gone: "The endpoint returned 410 Gone. Illarin has stopped announcing to it.",
  removed: "The integration was removed.",
  disabled: "The integration was switched off.",
  moved: "The integration moved to another address.",
  withheld: "Cancelled: this work is withheld.",
  withdrawn: "Cancelled: this update was withdrawn.",
  unlisted:
    "Cancelled: this work is unlisted and sharing its link was not approved.",
  deleted: "Cancelled: this work is no longer published.",
};

export function attemptStanding(one: Sending, now = new Date()): string {
  if (one.state === "delivered") {
    return `Delivered ${shortMoment(one.settledAt ?? one.occurredAt)}`;
  }
  if (one.state === "unconfirmed") {
    return "Discord did not confirm which message was posted. Check the channel before sending another announcement.";
  }
  if (one.state === "failed") {
    const stopped = STOPPED_WORDS[one.settledReason ?? ""];
    if (stopped) return stopped;
    if (one.settledReason === "exhausted") {
      return `Stopped after ${tries(one.tries)}`;
    }
    return `Refused. ${one.last?.detail ?? "The integration refused the announcement."}`;
  }
  if (one.tries === 0) return "Queued";
  const waited = new Date(one.dueAt).getTime() - now.getTime();
  if (waited <= 0) return "Sending";
  return `Attempt ${one.tries + 1} ${inWords(waited)}, after ${tries(one.tries)}`;
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
