import { expect, test } from "bun:test";
import {
  movedTo,
  postAddressIn,
  type WithdrawnPost,
} from "@/lib/publication-withdrawal";

const GONE: WithdrawnPost = {
  slug: "pictures-in-a-post",
  explanation: "We are clearing the rights on one of the pictures.",
};

test("a one-segment blog address is the address of a post", () => {
  expect(postAddressIn("/blog/pictures-in-a-post")).toBe("pictures-in-a-post");
  expect(postAddressIn("/blog/a%20post")).toBe("a post");
});

test("the feeds, the sitemap, the archives and the card are not posts", () => {
  expect(postAddressIn("/blog/feed.xml")).toBeNull();
  expect(postAddressIn("/blog/sitemap.xml")).toBeNull();
  expect(postAddressIn("/blog/category/announcement")).toBeNull();
  expect(postAddressIn("/blog/page/2")).toBeNull();
  expect(postAddressIn("/blog/a-post/card.png")).toBeNull();
  expect(postAddressIn("/blog/")).toBeNull();
  expect(postAddressIn("/browse")).toBeNull();
});

test("an address nobody could have typed reaches no post", () => {
  expect(postAddressIn("/blog/%E0%A4%A")).toBeNull();
});

test("a former address moves to the one the tombstone lives at", () => {
  expect(movedTo("pictures-in-an-article", GONE)).toBe(
    "/blog/pictures-in-a-post",
  );
  expect(movedTo("pictures-in-a-post", GONE)).toBeNull();
});
