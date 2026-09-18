import { expect, test } from "bun:test";
import type { WorkFollow } from "@/lib/api/notifications";
import {
  offersWatch,
  rememberNotNow,
  saidNotNow,
  watchWords,
} from "./asset-watch";

const ASSET_ID = "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a";

function watch(overrides: Partial<WorkFollow> = {}): WorkFollow {
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

test("an asset the reader has not watched offers to watch it", () => {
  expect(watchWords(watch(), "character")).toEqual({
    watching: false,
    name: "Not watching",
    detail: "Get a notification when this character updates.",
    action: "Watch",
  });
});

test("a watched asset says the reader hears when it updates", () => {
  expect(watchWords(watch({ state: "following" }), "lorebook")).toEqual({
    watching: true,
    name: "Watching",
    detail: "You get a notification when this lorebook updates.",
    action: "Stop watching",
  });
});

test("an installed asset names the instance that is the reason", () => {
  expect(
    watchWords(
      watch({ state: "installed", installedOn: ["Reading desk"] }),
      "theme",
    ),
  ).toEqual({
    watching: true,
    name: "Watching, installed on Reading desk",
    detail:
      "It is installed on Reading desk, so you get a notification when it updates.",
    action: "Stop watching",
  });
});

test("an asset installed on several instances names each of them", () => {
  const words = watchWords(
    watch({
      state: "installed",
      installedOn: ["Reading desk", "Studio", "Travel laptop"],
    }),
    "preset",
  );
  expect(words.name).toBe(
    "Watching, installed on Reading desk, Studio and Travel laptop",
  );
});

test("a stopped watch says an install does not start it again", () => {
  expect(
    watchWords(
      watch({ state: "stopped", installedOn: ["Reading desk"] }),
      "character",
    ),
  ).toEqual({
    watching: false,
    name: "Not watching",
    detail:
      "You stopped watching this character. Its install on Reading desk does not change that.",
    action: "Watch",
  });
  expect(watchWords(watch({ state: "stopped" }), "character").detail).toBe(
    "You stopped watching this character.",
  );
});

test("the offer appears only to a reader who has not watched, stopped or said not now", () => {
  expect(offersWatch(watch(), false)).toBe(true);
  expect(offersWatch(watch(), true)).toBe(false);
  expect(offersWatch(watch({ state: "following" }), false)).toBe(false);
  expect(
    offersWatch(
      watch({ state: "installed", installedOn: ["Reading desk"] }),
      false,
    ),
  ).toBe(false);
  expect(offersWatch(watch({ state: "stopped" }), false)).toBe(false);
  expect(offersWatch(undefined, false)).toBe(false);
});

test("not now is remembered for one asset in this browser", () => {
  const storage = memoryStorage();
  expect(saidNotNow(ASSET_ID, storage)).toBe(false);
  rememberNotNow(ASSET_ID, storage);
  expect(saidNotNow(ASSET_ID, storage)).toBe(true);
  expect(saidNotNow("7e2d9c1b-5a4f-4e3d-8c2b-1a0f9e8d7c6b", storage)).toBe(
    false,
  );
});

test("a browser that refuses storage asks again next time", () => {
  expect(() => rememberNotNow(ASSET_ID, refusingStorage)).not.toThrow();
  expect(saidNotNow(ASSET_ID, refusingStorage)).toBe(false);
});
