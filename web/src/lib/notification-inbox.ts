import type { Notification, NotificationList } from "@/lib/api/notifications";
import { assetHref } from "@/lib/asset-url";

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;
const BADGE_CEILING = 99;

const relative = new Intl.RelativeTimeFormat("en-GB", { numeric: "auto" });

const PROFILE_SETTINGS = "/settings/profile";

export type NotificationWords = {
  lead: string;
  subject: string;
  detail: string;
  href: string | null;
};

/** Says what an entry is about. A staff decision always reads as Illarin staff and never as the person who made it. */
export function notificationWords(entry: Notification): NotificationWords {
  const assetName = entry.asset?.name ?? "One of your assets";
  const assetPage = entry.asset
    ? assetHref(entry.asset.id, entry.asset.name)
    : null;
  switch (entry.type) {
    case "asset_withheld":
      return {
        lead: "Illarin staff withheld",
        subject: assetName,
        detail: entry.reason ?? "",
        href: assetPage,
      };
    case "asset_restored":
      return {
        lead: "Illarin staff restored",
        subject: assetName,
        detail: "Readers can reach it again.",
        href: assetPage,
      };
    case "profile_restricted":
      return {
        lead: "Illarin staff restricted",
        subject: "your profile",
        detail: entry.reason ?? "",
        href: PROFILE_SETTINGS,
      };
    case "profile_restored":
      return {
        lead: "Illarin staff restored",
        subject: "your profile",
        detail: "You can edit it again.",
        href: PROFILE_SETTINGS,
      };
  }
}

/** Says how long ago an entry arrived, in words for the past week and as a date before that. */
export function arrivedAgo(value: string, now: Date): string {
  const arrived = new Date(value);
  const elapsed = Math.max(0, now.getTime() - arrived.getTime());
  if (elapsed < MINUTE) return "just now";
  if (elapsed < HOUR)
    return relative.format(-Math.floor(elapsed / MINUTE), "minute");
  if (elapsed < DAY)
    return relative.format(-Math.floor(elapsed / HOUR), "hour");
  if (elapsed < WEEK) return relative.format(-Math.floor(elapsed / DAY), "day");
  return arrived.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: arrived.getFullYear() === now.getFullYear() ? undefined : "numeric",
  });
}

export function unreadBadge(count: number): string | null {
  if (count <= 0) return null;
  return count > BADGE_CEILING ? `${BADGE_CEILING}+` : String(count);
}

export function unreadLabel(count: number): string {
  return count > 0 ? `Notifications, ${count} unread` : "Notifications";
}

export function markedRead(
  page: NotificationList,
  id: string,
  at: string,
): NotificationList {
  return {
    ...page,
    items: page.items.map((item) =>
      item.id === id && !item.readAt ? { ...item, readAt: at } : item,
    ),
  };
}

export function allMarkedRead(
  page: NotificationList,
  at: string,
): NotificationList {
  return {
    ...page,
    items: page.items.map((item) =>
      item.readAt ? item : { ...item, readAt: at },
    ),
  };
}
