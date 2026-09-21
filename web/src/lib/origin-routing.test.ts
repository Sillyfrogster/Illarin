import { expect, test } from "bun:test";
import { type Asked, routeRequest } from "./origin-routing";
import type { UnpublishedPost } from "./post-unpublishing";

const nothingUnpublished = async () => null;

function unpublished(current: string, ...former: string[]) {
  const post: UnpublishedPost = { slug: current, explanation: "" };
  return async (slug: string) =>
    slug === current || former.includes(slug) ? post : null;
}

function asked(pathname: string): Asked {
  return { pathname };
}

test("an unpublished post answers gone under the blog", async () => {
  expect(
    await routeRequest(asked("/blog/gone-now"), unpublished("gone-now")),
  ).toEqual({ kind: "unpublished", slug: "gone-now" });
});

test("an unpublished post's former address goes to its current permalink", async () => {
  expect(
    await routeRequest(
      asked("/blog/old-name"),
      unpublished("gone-now", "old-name"),
    ),
  ).toEqual({
    kind: "redirect",
    to: "http://localhost:8000/blog/gone-now",
  });
});

test("only a single post address is checked for unpublishing", async () => {
  const lookedUp: string[] = [];
  const noting = async (slug: string) => {
    lookedUp.push(slug);
    return null;
  };
  for (const path of [
    "/",
    "/blog",
    "/blog/page/2",
    "/blog/first-post/card.png",
    "/blog/feed.xml",
  ]) {
    await routeRequest(asked(path), noting);
  }
  expect(lookedUp).toEqual([]);
  await routeRequest(asked("/blog/first-post"), noting);
  expect(lookedUp).toEqual(["first-post"]);
});

test("a published post and every non-blog page pass through", async () => {
  for (const path of ["/", "/browse", "/blog/first-post", "/posts"]) {
    expect(await routeRequest(asked(path), nothingUnpublished)).toEqual({
      kind: "pass",
    });
  }
});
