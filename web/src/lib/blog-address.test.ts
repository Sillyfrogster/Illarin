import { expect, test } from "bun:test";
import { blogAddress, blogHostname, postPermalink } from "./blog-address";

test("a permalink is the whole address a reader can hand to someone", () => {
  expect(postPermalink("first-post")).toBe(
    "http://blog.localhost:8000/first-post",
  );
});

test("a permalink escapes an address that needs it", () => {
  expect(postPermalink("café notes")).toBe(
    "http://blog.localhost:8000/caf%C3%A9%20notes",
  );
});

test("every blog address is on the blog origin, with its front page at the root", () => {
  expect(blogAddress("/")).toBe("http://blog.localhost:8000/");
  expect(blogAddress("/category/release/page/2")).toBe(
    "http://blog.localhost:8000/category/release/page/2",
  );
  expect(blogHostname).toBe("blog.localhost");
});
