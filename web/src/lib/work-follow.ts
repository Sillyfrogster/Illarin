import type { WorkFollow } from "@/lib/api/notifications";

type OfferStorage = Pick<Storage, "getItem" | "setItem">;

const NOT_NOW_PREFIX = "watch-offer-not-now:v1:";

const names = new Intl.ListFormat("en-GB", { type: "conjunction" });

export type FollowWords = {
  following: boolean;
  name: string;
  detail: string;
  action: string;
};

/** Says what following a work means for them, naming the instances when an install is the reason. */
export function followWords(follow: WorkFollow, typeName: string): FollowWords {
  const installedOn = names.format(follow.installedOn);
  switch (follow.state) {
    case "following":
      return {
        following: true,
        name: "Following",
        detail: `You get a notification when this ${typeName} updates.`,
        action: "Unfollow",
      };
    case "installed":
      return {
        following: true,
        name: `Following, installed on ${installedOn}`,
        detail: `It is installed on ${installedOn}, so you get a notification when it updates.`,
        action: "Unfollow",
      };
    case "stopped":
      return {
        following: false,
        name: "Not following",
        detail:
          follow.installedOn.length > 0
            ? `You unfollowed this ${typeName}. Its install on ${installedOn} does not change that.`
            : `You unfollowed this ${typeName}.`,
        action: "Follow",
      };
    case "none":
      return {
        following: false,
        name: "Not following",
        detail: `Get a notification when this ${typeName} updates.`,
        action: "Follow",
      };
  }
}

/** Says whether to offer to follow after a download or send. */
export function offersFollow(
  follow: WorkFollow | undefined,
  notNow: boolean,
): boolean {
  return follow?.state === "none" && !notNow;
}

/** Remembers "not now" in this browser only, because a refusal kept by Illarin would record the download. */
export function rememberNotNow(workId: string, storage?: OfferStorage): void {
  try {
    (storage ?? localStorage).setItem(NOT_NOW_PREFIX + workId, "1");
  } catch {}
}

export function saidNotNow(workId: string, storage?: OfferStorage): boolean {
  try {
    return (storage ?? localStorage).getItem(NOT_NOW_PREFIX + workId) === "1";
  } catch {
    return false;
  }
}
