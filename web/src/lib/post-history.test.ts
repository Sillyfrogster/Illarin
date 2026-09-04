import { describe, expect, test } from "bun:test";
import type { PostAction, PostRevision } from "@/lib/api/query";
import { historyStream, noteWords, revisionWords } from "./post-history";

function recorded(changes: Partial<PostAction>): PostAction {
  return {
    id: "a4c6a1f8-0000-4000-8000-000000000001",
    actor: "illarin.editor",
    credential: "session",
    action: "post.created",
    at: "2026-08-14T10:00:00Z",
    ...changes,
  };
}

function kept(changes: Partial<PostRevision>): PostRevision {
  return {
    id: "b7d2c3e4-0000-4000-8000-000000000001",
    number: 1,
    title: "Illarin ships weekly",
    summary: "What changed this week.",
    slug: "illarin-ships-weekly",
    category: {
      id: "c1",
      slug: "announcement",
      label: "Announcement",
      position: 0,
      retired: false,
    },
    capturedFor: "publication",
    capturedBy: "illarin.editor",
    capturedAt: "2026-08-14T11:00:00Z",
    public: true,
    ...changes,
  };
}

describe("historyStream", () => {
  test("orders editions and notes together, newest first", () => {
    const stream = historyStream(
      [kept({ capturedAt: "2026-08-14T11:00:00Z" })],
      [
        recorded({ action: "post.created", at: "2026-08-14T10:00:00Z" }),
        recorded({
          action: "post.address.corrected",
          at: "2026-08-14T12:00:00Z",
        }),
      ],
    );
    expect(stream.map((entry) => entry.kind)).toEqual([
      "note",
      "edition",
      "note",
    ]);
  });

  test("leaves out the actions an edition already accounts for", () => {
    const stream = historyStream(
      [kept({})],
      [
        recorded({ action: "post.published", revision: 1 }),
        recorded({ action: "post.checkpointed", revision: 1 }),
      ],
    );
    expect(stream).toHaveLength(1);
    expect(stream[0].kind).toBe("edition");
  });
});

describe("noteWords", () => {
  test("reads as a sentence after the name of who acted", () => {
    expect(noteWords(recorded({ action: "post.created" }))).toBe(
      "started this post",
    );
    expect(
      noteWords(recorded({ action: "post.revision.restored", revision: 2 })),
    ).toBe("restored edition 2");
    expect(noteWords(recorded({ action: "post.address.corrected" }))).toBe(
      "corrected the address",
    );
  });
});

describe("revisionWords", () => {
  test("separates a published edition from one kept while writing", () => {
    expect(revisionWords("publication")).toBe("Published");
    expect(revisionWords("checkpoint")).toBe("Kept while writing");
  });
});
