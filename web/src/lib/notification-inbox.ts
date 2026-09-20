import type { Notification, NotificationList } from "@/lib/api/notifications";
import { workHistoryHref, workHref } from "@/lib/work-url";

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
  const workName = entry.work?.name ?? "One of your works";
  const workPage = entry.work ? workHref(entry.work.id, entry.work.name) : null;
  switch (entry.type) {
    case "work_taken_down":
      return {
        lead: "Illarin staff took down",
        subject: workName,
        detail: entry.reason ?? "",
        href: workPage,
      };
    case "work_restored":
      return {
        lead: "Illarin staff restored",
        subject: workName,
        detail: "Readers can reach it again.",
        href: workPage,
      };
    case "work_updated":
      return {
        lead: updateLead(entry),
        subject: workName,
        detail: updateDetail(entry),
        href: entry.work
          ? workHistoryHref(
              entry.work.id,
              entry.work.name,
              entry.update?.number,
            )
          : null,
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

/** Says how many versions a folded entry stands for. */
function updateLead(entry: Notification): string {
  const arrivals = entry.update?.count ?? 1;
  return arrivals > 1 ? `${arrivals} new versions of` : "New version of";
}

function updateDetail(entry: Notification): string {
  const update = entry.update;
  if (!update) return "";
  const label = update.versionLabel ? `, ${update.versionLabel}` : "";
  return `Version ${update.number}${label}: ${update.summary}`;
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

export function removed(page: NotificationList, id: string): NotificationList {
  return { ...page, items: page.items.filter((item) => item.id !== id) };
}
