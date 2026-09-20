import { expect, test } from "bun:test";
import {
  archivePage,
  archivePath,
  BLOG_FEEDS,
  BLOG_HOME,
  feedAddresses,
  pageAddress,
  postPath,
} from "./blog-paths";

test("a post lives directly beneath the blog's front page", () => {
  expect(postPath("first-post")).toBe("/first-post");
  expect(postPath("café notes")).toBe("/caf%C3%A9%20notes");
});

test("a narrowed archive lives under its scope", () => {
  expect(archivePath("category", "release")).toBe("/category/release");
});

test("page one is the archive itself and later pages count from it", () => {
  expect(pageAddress(BLOG_HOME, 1)).toBe("/");
  expect(pageAddress(BLOG_HOME, 2)).toBe("/page/2");
  expect(pageAddress("/category/release", 1)).toBe("/category/release");
  expect(pageAddress("/category/release", 3)).toBe("/category/release/page/3");
});

test("a feed answers beneath the archive it summarizes", () => {
  expect(feedAddresses(BLOG_HOME)).toEqual({
    rss: "/feed.xml",
    json: "/feed.json",
  });
  expect(BLOG_FEEDS).toEqual({ rss: "/feed.xml", json: "/feed.json" });
  expect(feedAddresses("/category/release")).toEqual({
    rss: "/category/release/feed.xml",
    json: "/category/release/feed.json",
  });
});

test("an archive address with no page segment is page one", () => {
  expect(archivePage(undefined)).toBe(1);
  expect(archivePage([])).toBe(1);
});

test("a numbered archive address reads its page", () => {
  expect(archivePage(["page", "2"])).toBe(2);
  expect(archivePage(["page", "40"])).toBe(40);
});

test("anything but a whole page number counting from one is refused", () => {
  expect(archivePage(["page"])).toBeNull();
  expect(archivePage(["page", "0"])).toBeNull();
  expect(archivePage(["page", "-1"])).toBeNull();
  expect(archivePage(["page", "01"])).toBeNull();
  expect(archivePage(["page", "2.5"])).toBeNull();
  expect(archivePage(["page", "two"])).toBeNull();
  expect(archivePage(["archive", "2"])).toBeNull();
  expect(archivePage(["page", "2", "3"])).toBeNull();
});
