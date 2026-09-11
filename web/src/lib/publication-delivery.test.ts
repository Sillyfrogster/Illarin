import { describe, expect, test } from "bun:test";
import type { PostDelivery } from "@/lib/api/query";
import { deliveryStanding, deliveryState } from "@/lib/delivery-standing";
import {
  EVENT_WORDS,
  eventWord,
  transitionEvent,
} from "@/lib/publication-delivery";

const noon = new Date("2026-09-06T12:00:00Z");

function delivery(over: Partial<PostDelivery>): PostDelivery {
  return {
    id: "d",
    eventId: "e",
    eventType: "publication.post.published.v1",
    postId: "p",
    postTitle: "Illarin keeps its own writing now",
    revisionId: "r",
    destination: "Release feed",
    kind: "webhook",
    messageId: "",
    removed: false,
    state: "pending",
    run: 1,
    attempts: 0,
    occurredAt: "2026-09-06T11:00:00Z",
    dueAt: "2026-09-06T11:00:00Z",
    ...over,
  };
}

describe("what a delivery is doing", () => {
  test("names the four states a reader needs apart", () => {
    expect(deliveryState(delivery({ state: "pending" }))).toBe("waiting");
    expect(deliveryState(delivery({ state: "sending" }))).toBe("waiting");
    expect(
      deliveryState(delivery({ state: "delivered", settledReason: "arrived" })),
    ).toBe("arrived");
    expect(
      deliveryState(delivery({ state: "failed", settledReason: "exhausted" })),
    ).toBe("gaveUp");
    expect(
      deliveryState(delivery({ state: "failed", settledReason: "refused" })),
    ).toBe("gaveUp");
    expect(
      deliveryState(delivery({ state: "failed", settledReason: "disabled" })),
    ).toBe("stopped");
    expect(
      deliveryState(delivery({ state: "failed", settledReason: "removed" })),
    ).toBe("stopped");
  });

  test("says nothing has gone out yet before the first attempt", () => {
    expect(deliveryStanding(delivery({}), noon)).toBe("Queued");
  });

  test("says when the next attempt is due", () => {
    const waiting = delivery({
      attempts: 5,
      dueAt: "2026-09-06T16:00:00Z",
    });

    expect(deliveryStanding(waiting, noon)).toBe(
      "Attempt 6 in 4 hours, after 5 tries",
    );
  });

  test("rounds a wait under a minute to shortly", () => {
    const waiting = delivery({ attempts: 1, dueAt: "2026-09-06T12:00:20Z" });

    expect(deliveryStanding(waiting, noon)).toBe(
      "Attempt 2 shortly, after 1 try",
    );
  });

  test("says a due attempt is going out", () => {
    const due = delivery({ attempts: 1, dueAt: "2026-09-06T11:59:00Z" });

    expect(deliveryStanding(due, noon)).toBe("Sending");
  });

  test("says when it arrived", () => {
    const arrived = delivery({
      state: "delivered",
      settledReason: "arrived",
      attempts: 2,
      settledAt: "2026-09-06T11:30:00Z",
    });

    expect(deliveryStanding(arrived, noon)).toContain("Delivered");
  });

  test("says how many tries it gave up after", () => {
    const spent = delivery({
      state: "failed",
      settledReason: "exhausted",
      attempts: 10,
      settledAt: "2026-09-06T11:30:00Z",
    });

    expect(deliveryStanding(spent, noon)).toBe("Gave up after 10 tries");
  });

  test("says what turned it away", () => {
    const turned = delivery({
      state: "failed",
      settledReason: "refused",
      attempts: 1,
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

    expect(deliveryStanding(turned, noon)).toBe(
      "Delivery rejected. It answered 404.",
    );
  });

  test("says which configuration change stopped it", () => {
    for (const [reason, said] of [
      ["gone", "The endpoint returned 410 Gone. Delivery to it has stopped."],
      ["removed", "The destination was removed."],
      ["disabled", "The destination was switched off."],
      ["moved", "The destination moved to another address."],
    ] as const) {
      const stopped = delivery({
        state: "failed",
        settledReason: reason,
        attempts: 1,
        settledAt: "2026-09-06T11:30:00Z",
      });

      expect(deliveryStanding(stopped, noon)).toBe(said);
    }
  });
});

describe("the words for one public transition", () => {
  test("gives every event a short word and a sentence", () => {
    for (const event of [
      "publication.post.published.v1",
      "publication.post.updated.v1",
      "publication.post.withdrawn.v1",
    ] as const) {
      expect(EVENT_WORDS[event].word).not.toBe("");
      expect(EVENT_WORDS[event].what).not.toBe("");
    }
  });

  test("reads an event name off the wire", () => {
    expect(eventWord("publication.post.withdrawn.v1")).toBe("Withdrawn");
    expect(eventWord("publication.post.invented.v9")).toBe("Publication");
  });

  test("names the event each editor action sends", () => {
    expect(transitionEvent("publish")).toBe("publication.post.published.v1");
    expect(transitionEvent("changes")).toBe("publication.post.updated.v1");
    expect(transitionEvent("withdraw")).toBe("publication.post.withdrawn.v1");
    expect(transitionEvent("republish")).toBe("publication.post.published.v1");
  });
});
