import { ask } from "./request";
import type { components } from "./schema";

export type Notification = components["schemas"]["Notification"];
export type NotificationList = components["schemas"]["NotificationList"];
export type NotificationCursor = components["schemas"]["NotificationCursor"];
export type AssetWatch = components["schemas"]["AssetWatch"];

export const notificationKeys = {
  all: ["notifications"] as const,
  unread: (handle: string) => ["notifications", handle, "unread"] as const,
  inbox: (handle: string) => ["notifications", handle, "inbox"] as const,
};

const UNREADABLE = "Your notifications could not be loaded.";

/** Reads one page of the signed-in account's inbox, newest first. */
export async function readNotifications(
  limit: number,
  cursor?: NotificationCursor | null,
  signal?: AbortSignal,
): Promise<NotificationList> {
  const query = new URLSearchParams({ limit: String(limit) });
  if (cursor) {
    query.set("before", cursor.before);
    query.set("beforeId", cursor.beforeId);
  }
  const answer = await ask<NotificationList>(
    `/notifications?${query}`,
    { signal },
    (response) => response.json() as Promise<NotificationList>,
  );
  if (answer.value) return answer.value;
  throw new Error(answer.error ?? UNREADABLE);
}

export async function readUnreadCount(signal?: AbortSignal): Promise<number> {
  const answer = await ask<{ count: number }>(
    "/notifications/unread",
    { signal },
    (response) => response.json() as Promise<{ count: number }>,
  );
  if (answer.value) return answer.value.count;
  throw new Error(answer.error ?? UNREADABLE);
}

/** Marks one entry read, and keeps the request alive if the page moves on before it lands. */
export async function markNotificationRead(id: string): Promise<void> {
  const answer = await ask<null>(
    `/notifications/${encodeURIComponent(id)}/read`,
    { method: "POST", keepalive: true },
    async () => null,
  );
  if (answer.error) throw new Error(answer.error);
}

export function watchAsset(assetId: string): Promise<AssetWatch> {
  return changeWatch(assetId, "PUT");
}

export function stopWatchingAsset(assetId: string): Promise<AssetWatch> {
  return changeWatch(assetId, "DELETE");
}

async function changeWatch(
  assetId: string,
  method: "PUT" | "DELETE",
): Promise<AssetWatch> {
  const answer = await ask<AssetWatch>(
    `/assets/${encodeURIComponent(assetId)}/watch`,
    { method },
    (response) => response.json() as Promise<AssetWatch>,
  );
  if (answer.value) return answer.value;
  throw new Error(answer.error ?? "Your watch could not be changed.");
}

export async function markAllNotificationsRead(): Promise<void> {
  const answer = await ask<null>(
    "/notifications/read",
    { method: "POST" },
    async () => null,
  );
  if (answer.error) throw new Error(answer.error);
}
