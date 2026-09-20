import { describe, expect, test } from "bun:test";
import type {
  Post,
  PostDeletion,
  PostMedia,
  PostSchedule,
} from "@/lib/api/query";
import {
  draftFromPost,
  mergedMedia,
  postNotices,
  publishActions,
  publishLabel,
  savingWords,
  writerStanding,
} from "./post-writing";

function post(shape: Partial<Post>): Post {
  return {
    id: "post-1",
    status: "draft",
    title: "A title",
    summary: "",
    slug: "a-title",
    category: { id: "cat-1", label: "News", slug: "news" },
    body: { version: 1, content: [] },
    bodyVersion: 1,
    media: [],
    formerAddresses: [],
    version: 3,
    author: { handle: "writer" },
    createdAt: "2026-09-01T09:00:00Z",
    updatedAt: "2026-09-01T09:00:00Z",
    ...shape,
  } as Post;
}

function schedule(shape: Partial<PostSchedule>): PostSchedule {
  return {
    id: "sched-1",
    revisionId: "rev-2",
    revisionNumber: 2,
    at: "2026-09-12T08:10:00Z",
    state: "pending",
    createdBy: "writer",
    createdAt: "2026-09-10T08:10:00Z",
    ...shape,
  } as PostSchedule;
}

const deletion: PostDeletion = {
  at: "2026-09-06T15:23:00Z",
  until: "2026-10-06T15:23:00Z",
  by: "writer",
};

const unpublishing = {
  reason: "The numbers were wrong.",
  explanation: "",
  by: "editor",
  at: "2026-09-08T11:00:00Z",
};

describe("savingWords", () => {
  test("says what the post is when there is nothing to say about the save", () => {
    expect(savingWords("clean", post({ status: "published" }))).toBe(
      "Published",
    );
    expect(savingWords("clean", post({ status: "unpublished" }))).toBe(
      "Unpublished",
    );
    expect(savingWords("clean", post({}))).toBe("Saved privately");
  });

  test("a save in trouble outranks the post's own state", () => {
    const live = post({ status: "published" });
    expect(savingWords("conflict", live)).toBe("Someone else saved this post");
    expect(savingWords("refused", live)).toBe("Not saved");
    expect(savingWords("dirty", live)).toBe("Unsaved");
    expect(savingWords("saving", live)).toBe("Saving");
    expect(savingWords("saved", live)).toBe("Saved");
  });
});

describe("writerStanding", () => {
  test("a deleted post reads as deleted whatever readers last saw", () => {
    expect(writerStanding(post({ status: "published", deletion }))).toBe(
      "deleted",
    );
  });

  test("a schedule does not change what readers can do now", () => {
    expect(writerStanding(post({ schedule: schedule({}) }))).toBe("draft");
  });
});

describe("publishActions", () => {
  test("a draft can be published, scheduled and discarded", () => {
    expect(publishActions(post({}), false)).toEqual([
      "publish",
      "schedule",
      "delete",
    ]);
  });

  test("a published post publishes changes and comes down, and cannot be deleted", () => {
    expect(publishActions(post({ status: "published" }), true)).toEqual([
      "publish",
      "schedule",
      "unpublish",
    ]);
  });

  test("a unpublished post goes back up, and only an admin may delete it", () => {
    const down = post({ status: "unpublished", publishedAt: "2026-09-02" });
    expect(publishActions(down, false)).toEqual(["republish"]);
    expect(publishActions(down, true)).toEqual(["republish", "delete"]);
  });

  test("a deleted post offers recovery and nothing else", () => {
    expect(publishActions(post({ deletion }), true)).toEqual(["recover"]);
  });
});

describe("publishLabel", () => {
  test("names what the control is about to open", () => {
    expect(publishLabel(post({}))).toBe("Publish");
    expect(publishLabel(post({ status: "published" }))).toBe("Publish changes");
    expect(publishLabel(post({ status: "unpublished" }))).toBe(
      "Republish post",
    );
    expect(publishLabel(post({ deletion }))).toBe("Restore post");
  });
});

describe("postNotices", () => {
  test("says nothing about a plain draft", () => {
    expect(postNotices(post({}))).toEqual([]);
  });

  test("a deleted post says only that, and what it was deleted from", () => {
    const said = postNotices(
      post({ status: "unpublished", deletion, unpublishing }),
    );
    expect(said).toHaveLength(1);
    expect(said[0].kind).toBe("deleted");
    expect(said[0].tone).toBe("stop");
    expect(said[0].record).toBe("Deleted while unpublished.");
  });

  test("a unpublishing carries the record readers never see", () => {
    const said = postNotices(post({ status: "unpublished", unpublishing }));
    expect(said[0].record).toBe("The numbers were wrong.");
  });

  test("a unpublished post waiting to go live says both", () => {
    const said = postNotices(
      post({ status: "unpublished", unpublishing, schedule: schedule({}) }),
    );
    expect(said.map((one) => one.kind)).toEqual(["unpublished", "scheduled"]);
  });

  test("a stopped schedule is trouble and a waiting one is not", () => {
    const stopped = postNotices(
      post({ schedule: schedule({ state: "stopped" }) }),
    );
    expect(stopped[0].tone).toBe("stop");
    expect(postNotices(post({ schedule: schedule({}) }))[0].tone).toBe(
      "accent",
    );
  });

  test("a schedule that already ran or was called off says nothing", () => {
    expect(
      postNotices(post({ schedule: schedule({ state: "published" }) })),
    ).toEqual([]);
    expect(
      postNotices(post({ schedule: schedule({ state: "cancelled" }) })),
    ).toEqual([]);
  });
});

describe("draftFromPost", () => {
  test("carries the writable fields and leaves the rest behind", () => {
    const source = post({
      summary: "A summary",
      header: { mediaId: "m1", alt: "A picture" },
      linkCardMediaId: "m2",
    } as Partial<Post>);
    expect(draftFromPost(source)).toEqual({
      categoryId: "cat-1",
      title: "A title",
      summary: "A summary",
      slug: "a-title",
      body: { version: 1, content: [] },
      header: { mediaId: "m1", alt: "A picture", caption: "" },
      linkCardMediaId: "m2",
    });
  });

  test("a post with no picture carries none", () => {
    const made = draftFromPost(post({}));
    expect(made.header).toBeNull();
    expect(made.linkCardMediaId).toBeNull();
  });
});

describe("mergedMedia", () => {
  function media(id: string): PostMedia {
    return {
      id,
      url: `/${id}`,
      thumbUrl: `/${id}-t`,
      width: 2,
      height: 1,
    } as PostMedia;
  }

  test("keeps an upload the saved copy has not caught up with", () => {
    expect(
      mergedMedia([media("a"), media("b")], [media("a")]).map((one) => one.id),
    ).toEqual(["a", "b"]);
  });

  test("prefers the saved copy of a picture it already knows", () => {
    const saved = { ...media("a"), url: "/moved" };
    expect(mergedMedia([media("a")], [saved])[0].url).toBe("/moved");
  });
});
