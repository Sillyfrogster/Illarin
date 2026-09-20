import { expect, test } from "bun:test";
import { type Asked, routeRequest } from "./origin-routing";
import type { UnpublishedPost } from "./post-unpublishing";

const BLOG = "blog.localhost:8000";
const SITE = "localhost:8000";

const nothingUnpublished = async () => null;

function unpublished(current: string, ...former: string[]) {
  const post: UnpublishedPost = { slug: current, explanation: "" };
  return async (slug: string) =>
    slug === current || former.includes(slug) ? post : null;
}

function asked(host: string | null, pathname: string, search = ""): Asked {
  return { host, pathname, search };
}

test("the blog origin's front page is the blog tree's root", async () => {
  expect(await routeRequest(asked(BLOG, "/"), nothingUnpublished)).toEqual({
    kind: "rewrite",
    to: "/blog",
  });
});

test("every path on the blog origin is read inside the blog tree, query and all", async () => {
  for (const [path, inside] of [
    ["/page/2", "/blog/page/2"],
    ["/category/release", "/blog/category/release"],
    ["/app/lumiverse/feed.xml", "/blog/app/lumiverse/feed.xml"],
    ["/first-post", "/blog/first-post"],
    ["/first-post/card.png", "/blog/first-post/card.png"],
    ["/sitemap.xml", "/blog/sitemap.xml"],
    ["/robots.txt", "/blog/robots.txt"],
  ]) {
    expect(await routeRequest(asked(BLOG, path), nothingUnpublished)).toEqual({
      kind: "rewrite",
      to: inside,
    });
  }
  expect(
    await routeRequest(
      asked(BLOG, "/page/2", "?from=feed"),
      nothingUnpublished,
    ),
  ).toEqual({ kind: "rewrite", to: "/blog/page/2?from=feed" });
});

test("nothing on the blog origin escapes the blog tree, so the site's own pages never render there", async () => {
  for (const path of [
    "/sign-in",
    "/posts",
    "/browse",
    "/settings",
    "/upload",
    "/api/v1/auth/session",
    "/@someone",
  ]) {
    const route = await routeRequest(asked(BLOG, path), nothingUnpublished);
    expect(route).toEqual({ kind: "rewrite", to: `/blog${path}` });
  }
});

test("the blog origin still serves the application's own files and icons", async () => {
  for (const path of [
    "/_next/static/chunks/main.js",
    "/_next/image?url=x",
    "/favicon.ico",
    "/icon.svg",
    "/apple-icon.png",
    "/site-card.png",
    "/site.webmanifest",
    "/brand/illarin-horizontal-white.svg",
    "/brand/illarin-horizontal-black.svg",
    "/brand/icon-192.png",
    "/brand/app-icon-maskable.png",
  ]) {
    expect(await routeRequest(asked(BLOG, path), nothingUnpublished)).toEqual({
      kind: "pass",
    });
  }
});

test("an unpublished post's address answers gone, under the blog's own tombstone", async () => {
  expect(
    await routeRequest(asked(BLOG, "/gone-now"), unpublished("gone-now")),
  ).toEqual({ kind: "unpublished", slug: "gone-now" });
});

test("an unpublished post's former address goes straight to its current one", async () => {
  expect(
    await routeRequest(
      asked(BLOG, "/old-name"),
      unpublished("gone-now", "old-name"),
    ),
  ).toEqual({ kind: "redirect", to: "http://blog.localhost:8000/gone-now" });
});

test("only a single post address is checked for unpublishing", async () => {
  const lookedUp: string[] = [];
  const noting = async (slug: string) => {
    lookedUp.push(slug);
    return null;
  };
  for (const path of ["/", "/page/2", "/first-post/card.png", "/feed.xml"]) {
    await routeRequest(asked(BLOG, path), noting);
  }
  expect(lookedUp).toEqual([]);
  await routeRequest(asked(BLOG, "/first-post"), noting);
  expect(lookedUp).toEqual(["first-post"]);
});

test("the blog's former home on the main site sends readers to the blog origin for good", async () => {
  expect(await routeRequest(asked(SITE, "/blog"), nothingUnpublished)).toEqual({
    kind: "redirect",
    to: "http://blog.localhost:8000/",
  });
  expect(
    await routeRequest(
      asked(SITE, "/blog/first-post", "?ref=x"),
      nothingUnpublished,
    ),
  ).toEqual({
    kind: "redirect",
    to: "http://blog.localhost:8000/first-post?ref=x",
  });
  expect(
    await routeRequest(
      asked(SITE, "/blog/category/release/feed.xml"),
      nothingUnpublished,
    ),
  ).toEqual({
    kind: "redirect",
    to: "http://blog.localhost:8000/category/release/feed.xml",
  });
});

test("a doctored blog path never sends a reader off the blog host", async () => {
  for (const path of ["/blog//evil.com", "/blog/\\evil.com", "/blog//"]) {
    const route = await routeRequest(
      asked(SITE, path, "?x=1"),
      nothingUnpublished,
    );
    expect(route.kind).toBe("redirect");
    if (route.kind === "redirect") {
      expect(new URL(route.to).host).toBe(BLOG);
      expect(route.to.endsWith("?x=1")).toBe(true);
    }
  }
});

test("the main site is left alone everywhere else", async () => {
  for (const path of ["/", "/browse", "/blogger", "/api/v1/posts", "/posts"]) {
    expect(await routeRequest(asked(SITE, path), nothingUnpublished)).toEqual({
      kind: "pass",
    });
  }
});

test("the blog origin is known by its hostname whatever port or case it arrives with", async () => {
  for (const host of [
    "blog.localhost",
    "Blog.Localhost:8000",
    "blog.localhost:3000",
  ]) {
    expect(await routeRequest(asked(host, "/"), nothingUnpublished)).toEqual({
      kind: "rewrite",
      to: "/blog",
    });
  }
  for (const host of [null, "", "localhost:3000", "notblog.localhost:8000"]) {
    expect(await routeRequest(asked(host, "/"), nothingUnpublished)).toEqual({
      kind: "pass",
    });
  }
});
