import { expect, test } from "bun:test";
import { blogAddress, postPermalink } from "./blog-address";

test("a permalink is the whole address a reader can hand to someone", () => {
  expect(postPermalink("first-post")).toBe(
    "http://localhost:8000/blog/first-post",
  );
});

test("a permalink escapes an address that needs it", () => {
  expect(postPermalink("café notes")).toBe(
    "http://localhost:8000/blog/caf%C3%A9%20notes",
  );
});

test("every blog address is under the site origin", () => {
  expect(blogAddress("/blog")).toBe("http://localhost:8000/blog");
  expect(blogAddress("/blog/category/release/page/2")).toBe(
    "http://localhost:8000/blog/category/release/page/2",
  );
});
