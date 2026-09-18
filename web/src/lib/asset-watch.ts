import type { WorkFollow } from "@/lib/api/notifications";

type OfferStorage = Pick<Storage, "getItem" | "setItem">;

const NOT_NOW_PREFIX = "watch-offer-not-now:v1:";

const names = new Intl.ListFormat("en-GB", { type: "conjunction" });

export type WatchWords = {
  watching: boolean;
  name: string;
  detail: string;
  action: string;
};

/** Says what a reader's watch means for them, naming the instances when an install is the reason. */
export function watchWords(watch: WorkFollow, kind: string): WatchWords {
  const installedOn = names.format(watch.installedOn);
  switch (watch.state) {
    case "watching":
      return {
        watching: true,
        name: "Watching",
        detail: `You get a notification when this ${kind} updates.`,
        action: "Stop watching",
      };
    case "installed":
      return {
        watching: true,
        name: `Watching, installed on ${installedOn}`,
        detail: `It is installed on ${installedOn}, so you get a notification when it updates.`,
        action: "Stop watching",
      };
    case "stopped":
      return {
        watching: false,
        name: "Not watching",
        detail:
          watch.installedOn.length > 0
            ? `You stopped watching this ${kind}. Its install on ${installedOn} does not change that.`
            : `You stopped watching this ${kind}.`,
        action: "Watch",
      };
    case "none":
      return {
        watching: false,
        name: "Not watching",
        detail: `Get a notification when this ${kind} updates.`,
        action: "Watch",
      };
  }
}

/** Says whether to offer a watch after a download or send. */
export function offersWatch(
  watch: WorkFollow | undefined,
  notNow: boolean,
): boolean {
  return watch?.state === "none" && !notNow;
}

/** Remembers "not now" in this browser only, because a refusal kept by Illarin would record the download. */
export function rememberNotNow(assetId: string, storage?: OfferStorage): void {
  try {
    (storage ?? localStorage).setItem(NOT_NOW_PREFIX + assetId, "1");
  } catch {}
}

export function saidNotNow(assetId: string, storage?: OfferStorage): boolean {
  try {
    return (storage ?? localStorage).getItem(NOT_NOW_PREFIX + assetId) === "1";
  } catch {
    return false;
  }
}
