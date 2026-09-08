import { expect, test } from "bun:test";
import type { PostSummary } from "@/lib/api/query";
import { entryFacts } from "./archive-entry";

const ILLARIN = {
  id: "9d3f1c00-0000-4000-8000-000000000001",
  slug: "illarin",
  name: "Illarin",
  home: "https://illarin.xyz",
  position: 0,
  retired: false,
  destinations: [],
};

const LUMIVERSE = { ...ILLARIN, slug: "lumiverse", name: "Lumiverse" };

const POST: PostSummary = {
  id: "0f1e2d3c-4b5a-4968-8776-554433221100",
  slug: "a-post",
  originalSlug: "a-post",
  title: "A post",
  summary: "One line about it.",
  category: {
    id: "1",
    slug: "announcement",
    label: "Announcement",
    position: 0,
    retired: false,
  },
  byline: {
    handle: "wren",
    historical: false,
    displayName: "Wren Ashdown",
    contactEmail: "",
    positions: [],
    distinctions: [],
  },
  publishedAt: "2026-09-06T10:00:00Z",
};

test("a post with no app names no app", () => {
  expect(entryFacts(POST, null).app).toBeNull();
});

test("an app the byline already stands for is not named twice", () => {
  const post = {
    ...POST,
    app: ILLARIN,
    byline: { ...POST.byline, app: ILLARIN },
  };
  expect(entryFacts(post, null).app).toBeNull();
  expect(entryFacts(post, null).affiliation).toBe(true);
});

test("a release names its app even when the byline stands for the same one", () => {
  const post = {
    ...POST,
    app: ILLARIN,
    releaseVersion: "2.1",
    byline: { ...POST.byline, app: ILLARIN },
  };
  expect(entryFacts(post, null).app).toEqual(ILLARIN);
  expect(entryFacts(post, null).affiliation).toBe(false);
});

test("an app the byline does not stand for is named", () => {
  const post = {
    ...POST,
    app: LUMIVERSE,
    byline: { ...POST.byline, app: ILLARIN },
  };
  expect(entryFacts(post, null).app).toEqual(LUMIVERSE);
  expect(entryFacts(post, null).affiliation).toBe(true);
});

test("an app archive states its app once, in its heading, and never in a row", () => {
  const post = { ...POST, app: ILLARIN, releaseVersion: "2.1" };
  const facts = entryFacts(post, "app");
  expect(facts.app).toBeNull();
  expect(facts.affiliation).toBe(false);
});

test("a category archive states its category once, in its heading", () => {
  expect(entryFacts(POST, "category").category).toBe(false);
  expect(entryFacts(POST, null).category).toBe(true);
  expect(entryFacts(POST, "app").category).toBe(true);
});
