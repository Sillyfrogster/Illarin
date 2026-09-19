import { expect, test } from "bun:test";
import type { Notification, NotificationList } from "@/lib/api/notifications";
import {
  allMarkedRead,
  arrivedAgo,
  markedRead,
  notificationWords,
  removed,
  unreadBadge,
  unreadLabel,
} from "./notification-inbox";

const WORK = {
  id: "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a",
  name: "Moonlit Archive",
};
const NOW = new Date("2026-09-14T12:00:00Z");

function entry(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "8c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f",
    type: "work_withheld",
    createdAt: "2026-09-14T09:00:00Z",
    work: WORK,
    reason: "Copyright report under review",
    ...overrides,
  };
}

function before(milliseconds: number): string {
  return new Date(NOW.getTime() - milliseconds).toISOString();
}

test("a withheld work says Illarin staff withheld it, gives the reason and opens the work", () => {
  expect(notificationWords(entry())).toEqual({
    lead: "Illarin staff withheld",
    subject: "Moonlit Archive",
    detail: "Copyright report under review",
    href: "/a/0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a/moonlit-archive",
  });
});

test("a restored work says Illarin staff restored it and that readers can reach it again", () => {
  expect(
    notificationWords(entry({ type: "work_restored", reason: undefined })),
  ).toEqual({
    lead: "Illarin staff restored",
    subject: "Moonlit Archive",
    detail: "Readers can reach it again.",
    href: "/a/0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a/moonlit-archive",
  });
});

test("a restricted profile says Illarin staff restricted it, gives the reason and opens the profile settings", () => {
  expect(
    notificationWords(
      entry({
        type: "profile_restricted",
        work: undefined,
        reason: "Impersonating another creator",
      }),
    ),
  ).toEqual({
    lead: "Illarin staff restricted",
    subject: "your profile",
    detail: "Impersonating another creator",
    href: "/settings/profile",
  });
});

test("a restored profile says Illarin staff restored it and that it can be edited again", () => {
  expect(
    notificationWords(
      entry({ type: "profile_restored", work: undefined, reason: undefined }),
    ),
  ).toEqual({
    lead: "Illarin staff restored",
    subject: "your profile",
    detail: "You can edit it again.",
    href: "/settings/profile",
  });
});

test("an updated work names the update, its version and summary, and opens that update in the history", () => {
  expect(
    notificationWords(
      entry({
        type: "work_updated",
        reason: undefined,
        update: {
          number: 3,
          count: 1,
          versionLabel: "v2.1",
          summary: "Rewrote her opening",
        },
      }),
    ),
  ).toEqual({
    lead: "New version of",
    subject: "Moonlit Archive",
    detail: "Version 3, v2.1: Rewrote her opening",
    href: "/a/0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a/moonlit-archive?history#version-3",
  });
});

test("an update without a version label leaves the label out", () => {
  expect(
    notificationWords(
      entry({
        type: "work_updated",
        reason: undefined,
        update: {
          number: 2,
          count: 1,
          summary: "Fixed a typo in her greeting",
        },
      }),
    ).detail,
  ).toBe("Version 2: Fixed a typo in her greeting");
});

test("a folded entry says how many updates arrived and still shows the latest", () => {
  const words = notificationWords(
    entry({
      type: "work_updated",
      reason: undefined,
      update: {
        number: 7,
        count: 5,
        versionLabel: "v3",
        summary: "Added a third greeting",
      },
    }),
  );
  expect(words.lead).toBe("5 new versions of");
  expect(words.subject).toBe("Moonlit Archive");
  expect(words.detail).toBe("Version 7, v3: Added a third greeting");
  expect(words.href).toBe(
    "/a/0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a/moonlit-archive?history#version-7",
  );
});

test("an entry holding two updates counts them and one holding a single update does not", () => {
  const folded = (count: number) =>
    notificationWords(
      entry({
        type: "work_updated",
        reason: undefined,
        update: { number: 4, count, summary: "Rewrote her opening" },
      }),
    ).lead;
  expect(folded(1)).toBe("New version of");
  expect(folded(2)).toBe("2 new versions of");
});

test("an entry that names no work still reads plainly and has nowhere to send the reader", () => {
  const words = notificationWords(entry({ work: undefined }));
  expect(words.subject).toBe("One of your works");
  expect(words.href).toBeNull();
});

test("arrival reads in words for the last week and as a date after that", () => {
  expect(arrivedAgo(before(30_000), NOW)).toBe("just now");
  expect(arrivedAgo(before(60_000), NOW)).toBe("1 minute ago");
  expect(arrivedAgo(before(5 * 60_000), NOW)).toBe("5 minutes ago");
  expect(arrivedAgo(before(3 * 3_600_000), NOW)).toBe("3 hours ago");
  expect(arrivedAgo(before(26 * 3_600_000), NOW)).toBe("yesterday");
  expect(arrivedAgo(before(3 * 86_400_000), NOW)).toBe("3 days ago");
  expect(arrivedAgo(before(10 * 86_400_000), NOW)).toBe("4 September");
  expect(arrivedAgo("2025-11-02T12:00:00Z", NOW)).toBe("2 November 2025");
});

test("an arrival stamped a moment ahead of this clock still reads as just now", () => {
  expect(arrivedAgo(new Date(NOW.getTime() + 4_000).toISOString(), NOW)).toBe(
    "just now",
  );
});

test("the badge shows nothing at zero and stops counting past 99", () => {
  expect(unreadBadge(0)).toBeNull();
  expect(unreadBadge(7)).toBe("7");
  expect(unreadBadge(99)).toBe("99");
  expect(unreadBadge(100)).toBe("99+");
});

test("the bell names how many notifications are unread", () => {
  expect(unreadLabel(0)).toBe("Notifications");
  expect(unreadLabel(1)).toBe("Notifications, 1 unread");
  expect(unreadLabel(140)).toBe("Notifications, 140 unread");
});

test("opening an entry marks only that entry read and leaves the page it came from alone", () => {
  const opened = entry();
  const other = entry({ id: "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d" });
  const page: NotificationList = { items: [opened, other] };

  const marked = markedRead(page, opened.id, "2026-09-14T12:00:00Z");

  expect(marked.items.map((item) => item.readAt)).toEqual([
    "2026-09-14T12:00:00Z",
    undefined,
  ]);
  expect(page.items[0].readAt).toBeUndefined();
});

test("marking read keeps the time an entry was first opened", () => {
  const page: NotificationList = {
    items: [entry({ readAt: "2026-09-13T08:00:00Z" })],
  };
  expect(
    markedRead(page, page.items[0].id, "2026-09-14T12:00:00Z").items[0].readAt,
  ).toBe("2026-09-13T08:00:00Z");
});

test("marking everything read stamps each unread entry and keeps the cursor", () => {
  const cursor = { before: "2026-09-10T00:00:00Z", beforeId: WORK.id };
  const page: NotificationList = {
    items: [entry(), entry({ id: "x", readAt: "2026-09-13T08:00:00Z" })],
    nextCursor: cursor,
  };

  expect(allMarkedRead(page, "2026-09-14T12:00:00Z")).toEqual({
    items: [
      entry({ readAt: "2026-09-14T12:00:00Z" }),
      entry({ id: "x", readAt: "2026-09-13T08:00:00Z" }),
    ],
    nextCursor: cursor,
  });
});

test("removing an entry takes only that entry out of the page and keeps the cursor", () => {
  const gone = entry();
  const kept = entry({ id: "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d" });
  const cursor = { before: "2026-09-10T00:00:00Z", beforeId: WORK.id };
  const page: NotificationList = { items: [gone, kept], nextCursor: cursor };

  expect(removed(page, gone.id)).toEqual({ items: [kept], nextCursor: cursor });
  expect(page.items).toHaveLength(2);
});
