import { describe, expect, test } from "bun:test";
import type { BlogAnnouncementAttempt } from "@/lib/api/query";
import { attemptStanding, attemptState } from "@/lib/attempt-standing";
import {
  ANNOUNCEMENT_WORDS,
  announcementWord,
  transitionEvent,
} from "@/lib/blog-announcement-attempt";

const noon = new Date("2026-09-06T12:00:00Z");

function delivery(
  over: Partial<BlogAnnouncementAttempt>,
): BlogAnnouncementAttempt {
  return {
    id: "d",
    announcementId: "e",
    announcementType: "blog.post.published.v1",
    postId: "p",
    postTitle: "Illarin keeps its own writing now",
    revisionId: "r",
    integration: "Release feed",
    type: "webhook",
    messageId: "",
    removed: false,
    state: "pending",
    run: 1,
    tries: 0,
    occurredAt: "2026-09-06T11:00:00Z",
    dueAt: "2026-09-06T11:00:00Z",
    ...over,
  };
}

describe("what a delivery is doing", () => {
  test("names the four states a reader needs apart", () => {
    expect(attemptState(delivery({ state: "pending" }))).toBe("waiting");
    expect(attemptState(delivery({ state: "sending" }))).toBe("waiting");
    expect(
      attemptState(delivery({ state: "delivered", settledReason: "arrived" })),
    ).toBe("arrived");
    expect(
      attemptState(delivery({ state: "failed", settledReason: "exhausted" })),
    ).toBe("gaveUp");
    expect(
      attemptState(delivery({ state: "failed", settledReason: "refused" })),
    ).toBe("gaveUp");
    expect(
      attemptState(delivery({ state: "failed", settledReason: "disabled" })),
    ).toBe("stopped");
    expect(
      attemptState(delivery({ state: "failed", settledReason: "removed" })),
    ).toBe("stopped");
  });

  test("says nothing has gone out yet before the first attempt", () => {
    expect(attemptStanding(delivery({}), noon)).toBe("Queued");
  });

  test("says when the next attempt is due", () => {
    const waiting = delivery({
      tries: 5,
      dueAt: "2026-09-06T16:00:00Z",
    });

    expect(attemptStanding(waiting, noon)).toBe(
      "Attempt 6 in 4 hours, after 5 tries",
    );
  });

  test("rounds a wait under a minute to shortly", () => {
    const waiting = delivery({ tries: 1, dueAt: "2026-09-06T12:00:20Z" });

    expect(attemptStanding(waiting, noon)).toBe(
      "Attempt 2 shortly, after 1 try",
    );
  });

  test("says a due attempt is going out", () => {
    const due = delivery({ tries: 1, dueAt: "2026-09-06T11:59:00Z" });

    expect(attemptStanding(due, noon)).toBe("Sending");
  });

  test("says when it arrived", () => {
    const arrived = delivery({
      state: "delivered",
      settledReason: "arrived",
      tries: 2,
      settledAt: "2026-09-06T11:30:00Z",
    });

    expect(attemptStanding(arrived, noon)).toContain("Delivered");
  });

  test("says how many tries it stopped after", () => {
    const spent = delivery({
      state: "failed",
      settledReason: "exhausted",
      tries: 10,
      settledAt: "2026-09-06T11:30:00Z",
    });

    expect(attemptStanding(spent, noon)).toBe("Stopped after 10 tries");
  });

  test("says what turned it away", () => {
    const turned = delivery({
      state: "failed",
      settledReason: "refused",
      tries: 1,
      settledAt: "2026-09-06T11:30:00Z",
      last: {
        run: 1,
        number: 1,
        outcome: "refused",
        detail: "It answered 404.",
        tookMs: 40,
        attemptedAt: "2026-09-06T11:30:00Z",
      },
    });

    expect(attemptStanding(turned, noon)).toBe("Refused. It answered 404.");
  });

  test("says which configuration change stopped it", () => {
    for (const [reason, said] of [
      [
        "gone",
        "The endpoint returned 410 Gone. Illarin has stopped announcing to it.",
      ],
      ["removed", "The integration was removed."],
      ["disabled", "The integration was switched off."],
      ["moved", "The integration moved to another address."],
    ] as const) {
      const stopped = delivery({
        state: "failed",
        settledReason: reason,
        tries: 1,
        settledAt: "2026-09-06T11:30:00Z",
      });

      expect(attemptStanding(stopped, noon)).toBe(said);
    }
  });
});

describe("the words for one public transition", () => {
  test("gives every event a short word and a sentence", () => {
    for (const announcement of [
      "blog.post.published.v1",
      "blog.post.updated.v1",
      "blog.post.unpublished.v1",
    ] as const) {
      expect(ANNOUNCEMENT_WORDS[announcement].word).not.toBe("");
      expect(ANNOUNCEMENT_WORDS[announcement].what).not.toBe("");
    }
  });

  test("reads an event name off the wire", () => {
    expect(announcementWord("blog.post.unpublished.v1")).toBe("Unpublished");
    expect(announcementWord("blog.post.invented.v9")).toBe("Publication");
  });

  test("names the event each editor action sends", () => {
    expect(transitionEvent("publish")).toBe("blog.post.published.v1");
    expect(transitionEvent("changes")).toBe("blog.post.updated.v1");
    expect(transitionEvent("unpublish")).toBe("blog.post.unpublished.v1");
    expect(transitionEvent("republish")).toBe("blog.post.published.v1");
  });
});
