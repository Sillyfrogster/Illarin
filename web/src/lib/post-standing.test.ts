import { describe, expect, test } from "bun:test";
import type { Post, PostSchedule } from "@/lib/api/query";
import { goingLiveAt, inStanding, lifecycleOf } from "./post-standing";

function post(shape: Partial<Post>): Post {
  return { status: "draft", ...shape } as Post;
}

const waiting = {
  state: "pending",
  at: "2026-09-12T08:10:00Z",
} as PostSchedule;

describe("lifecycleOf", () => {
  test("calls a deleted post deleted whatever readers last saw", () => {
    const deleted = post({
      status: "withdrawn",
      deletion: {
        at: "2026-09-06T15:23:00Z",
        until: "2026-10-06T15:23:00Z",
        by: "writer",
      },
    });
    expect(lifecycleOf(deleted)).toBe("deleted");
  });

  test("reads the public state of a live post", () => {
    expect(lifecycleOf(post({ status: "published" }))).toBe("published");
    expect(lifecycleOf(post({ status: "withdrawn" }))).toBe("withdrawn");
    expect(lifecycleOf(post({ status: "draft" }))).toBe("draft");
  });
});

describe("goingLiveAt", () => {
  test("answers the instant an edition is still on its way", () => {
    expect(goingLiveAt(post({ schedule: waiting }))).toBe(
      "2026-09-12T08:10:00Z",
    );
  });

  test("answers nothing for a schedule that has settled", () => {
    const settled = { ...waiting, state: "published" } as PostSchedule;
    expect(goingLiveAt(post({ schedule: settled }))).toBeNull();
    expect(goingLiveAt(post({}))).toBeNull();
  });
});

describe("inStanding", () => {
  test("puts a published post with an edition waiting in both standings", () => {
    const both = post({ status: "published", schedule: waiting });
    expect(inStanding(both, "published")).toBe(true);
    expect(inStanding(both, "scheduled")).toBe(true);
  });

  test("keeps scheduling out of the lifecycle standings", () => {
    const draft = post({ status: "draft", schedule: waiting });
    expect(inStanding(draft, "draft")).toBe(true);
    expect(inStanding(draft, "published")).toBe(false);
  });

  test("takes every post in the list it was given for everything and deleted", () => {
    expect(inStanding(post({ status: "published" }), "everything")).toBe(true);
    expect(inStanding(post({ status: "withdrawn" }), "deleted")).toBe(true);
  });
});
