import { expect, test } from "bun:test";
import type { WorkFollow } from "@/lib/api/notifications";
import {
  followWords,
  offersFollow,
  rememberNotNow,
  saidNotNow,
} from "./work-follow";

const WORK_ID = "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a";

function follow(overrides: Partial<WorkFollow> = {}): WorkFollow {
  return { state: "none", installedOn: [], ...overrides };
}

function memoryStorage(): Pick<Storage, "getItem" | "setItem"> {
  const kept = new Map<string, string>();
  return {
    getItem: (key) => kept.get(key) ?? null,
    setItem: (key, value) => {
      kept.set(key, value);
    },
  };
}

const refusingStorage: Pick<Storage, "getItem" | "setItem"> = {
  getItem: () => {
    throw new Error("storage is off");
  },
  setItem: () => {
    throw new Error("storage is off");
  },
};

test("a work the reader does not follow offers to follow it", () => {
  expect(followWords(follow(), "character")).toEqual({
    following: false,
    name: "Not following",
    detail: "Get a notification when this character updates.",
    action: "Follow",
  });
});

test("a followed work says the reader hears when it updates", () => {
  expect(followWords(follow({ state: "following" }), "lorebook")).toEqual({
    following: true,
    name: "Following",
    detail: "You get a notification when this lorebook updates.",
    action: "Unfollow",
  });
});

test("an installed work names the instance that is the reason", () => {
  expect(
    followWords(
      follow({ state: "installed", installedOn: ["Reading desk"] }),
      "theme",
    ),
  ).toEqual({
    following: true,
    name: "Following, installed on Reading desk",
    detail:
      "It is installed on Reading desk, so you get a notification when it updates.",
    action: "Unfollow",
  });
});

test("a work installed on several instances names each of them", () => {
  const words = followWords(
    follow({
      state: "installed",
      installedOn: ["Reading desk", "Studio", "Travel laptop"],
    }),
    "preset",
  );
  expect(words.name).toBe(
    "Following, installed on Reading desk, Studio and Travel laptop",
  );
});

test("an unfollow says an install does not start following again", () => {
  expect(
    followWords(
      follow({ state: "stopped", installedOn: ["Reading desk"] }),
      "character",
    ),
  ).toEqual({
    following: false,
    name: "Not following",
    detail:
      "You unfollowed this character. Its install on Reading desk does not change that.",
    action: "Follow",
  });
  expect(followWords(follow({ state: "stopped" }), "character").detail).toBe(
    "You unfollowed this character.",
  );
});

test("the offer appears only to a reader who has not followed, unfollowed or said not now", () => {
  expect(offersFollow(follow(), false)).toBe(true);
  expect(offersFollow(follow(), true)).toBe(false);
  expect(offersFollow(follow({ state: "following" }), false)).toBe(false);
  expect(
    offersFollow(
      follow({ state: "installed", installedOn: ["Reading desk"] }),
      false,
    ),
  ).toBe(false);
  expect(offersFollow(follow({ state: "stopped" }), false)).toBe(false);
  expect(offersFollow(undefined, false)).toBe(false);
});

test("not now is remembered for one work in this browser", () => {
  const storage = memoryStorage();
  expect(saidNotNow(WORK_ID, storage)).toBe(false);
  rememberNotNow(WORK_ID, storage);
  expect(saidNotNow(WORK_ID, storage)).toBe(true);
  expect(saidNotNow("7e2d9c1b-5a4f-4e3d-8c2b-1a0f9e8d7c6b", storage)).toBe(
    false,
  );
});

test("a browser that refuses storage asks again next time", () => {
  expect(() => rememberNotNow(WORK_ID, refusingStorage)).not.toThrow();
  expect(saidNotNow(WORK_ID, refusingStorage)).toBe(false);
});
