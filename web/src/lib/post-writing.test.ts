import { describe, expect, test } from "bun:test";
import type {
  Post,
  PostDeletion,
  PostMedia,
  PostSchedule,
  PublicationApp,
} from "@/lib/api/query";
import {
  draftFromPost,
  mergedMedia,
  namedRelease,
  postNotices,
  publicationActions,
  publicationLabel,
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
    document: { version: 1, content: [] },
    documentVersion: 1,
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

const withdrawal = {
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
    expect(savingWords("clean", post({ status: "withdrawn" }))).toBe(
      "Withdrawn",
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

describe("publicationActions", () => {
  test("a draft can be published, scheduled and discarded", () => {
    expect(publicationActions(post({}), false)).toEqual([
      "publish",
      "schedule",
      "delete",
    ]);
  });

  test("a published post publishes changes and comes down, and cannot be deleted", () => {
    expect(publicationActions(post({ status: "published" }), true)).toEqual([
      "publish",
      "schedule",
      "withdraw",
    ]);
  });

  test("a withdrawn post goes back up, and only an admin may delete it", () => {
    const down = post({ status: "withdrawn", publishedAt: "2026-09-02" });
    expect(publicationActions(down, false)).toEqual(["republish"]);
    expect(publicationActions(down, true)).toEqual(["republish", "delete"]);
  });

  test("a deleted post offers recovery and nothing else", () => {
    expect(publicationActions(post({ deletion }), true)).toEqual(["recover"]);
  });
});

describe("publicationLabel", () => {
  test("names what the control is about to open", () => {
    expect(publicationLabel(post({}))).toBe("Publish");
    expect(publicationLabel(post({ status: "published" }))).toBe(
      "Publish changes",
    );
    expect(publicationLabel(post({ status: "withdrawn" }))).toBe(
      "Republish post",
    );
    expect(publicationLabel(post({ deletion }))).toBe("Restore post");
  });
});

describe("postNotices", () => {
  test("says nothing about a plain draft", () => {
    expect(postNotices(post({}))).toEqual([]);
  });

  test("a deleted post says only that, and what it was deleted from", () => {
    const said = postNotices(
      post({ status: "withdrawn", deletion, withdrawal }),
    );
    expect(said).toHaveLength(1);
    expect(said[0].kind).toBe("deleted");
    expect(said[0].tone).toBe("stop");
    expect(said[0].record).toBe("Deleted while withdrawn.");
  });

  test("a withdrawal carries the record readers never see", () => {
    const said = postNotices(post({ status: "withdrawn", withdrawal }));
    expect(said[0].record).toBe("The numbers were wrong.");
  });

  test("a withdrawn post waiting to go live says both", () => {
    const said = postNotices(
      post({ status: "withdrawn", withdrawal, schedule: schedule({}) }),
    );
    expect(said.map((one) => one.kind)).toEqual(["withdrawn", "scheduled"]);
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
      socialMediaId: "m2",
      release: {
        app: {
          id: "app-1",
          name: "Illarin",
          slug: "illarin",
          home: "https://one",
        },
        version: "2.4.0",
      },
    } as Partial<Post>);
    expect(draftFromPost(source)).toEqual({
      categoryId: "cat-1",
      title: "A title",
      summary: "A summary",
      slug: "a-title",
      document: { version: 1, content: [] },
      header: { mediaId: "m1", alt: "A picture", caption: "" },
      socialMediaId: "m2",
      release: { appId: "app-1", version: "2.4.0", address: "" },
    });
  });

  test("a post with no picture and no release carries neither", () => {
    const made = draftFromPost(post({}));
    expect(made.header).toBeNull();
    expect(made.release).toBeNull();
    expect(made.socialMediaId).toBeNull();
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

describe("namedRelease", () => {
  const open = [
    { id: "app-1", name: "Illarin", slug: "illarin", home: "https://one" },
  ] as PublicationApp[];

  test("keeps a retired project listed while a post still names it", () => {
    const retired = {
      id: "app-9",
      name: "Retired",
      slug: "retired",
      home: "https://nine",
    } as PublicationApp;
    const listed = namedRelease(
      open,
      post({ release: { app: retired, version: "1" } } as Partial<Post>),
    );
    expect(listed.map((one) => one.id)).toEqual(["app-9", "app-1"]);
  });

  test("does not list a project twice", () => {
    const naming = post({
      release: { app: open[0], version: "1" },
    } as Partial<Post>);
    expect(namedRelease(open, naming)).toHaveLength(1);
  });
});
