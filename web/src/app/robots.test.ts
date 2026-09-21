import { expect, test } from "bun:test";
import { GET } from "./blog/robots.txt/route";
import { buildRobots } from "./robots";

const robots = buildRobots("https://illarin.com");
const rules = Array.isArray(robots.rules) ? robots.rules[0] : robots.rules;
const disallowed = [rules.disallow ?? []].flat();

test("points crawlers at the site and blog sitemaps", () => {
  expect(robots.sitemap).toEqual([
    "https://illarin.com/sitemap.xml",
    "https://illarin.com/blog/sitemap.xml",
  ]);
});

test("the blog robots route uses the site's address", async () => {
  expect(await GET().text()).toBe(
    "User-Agent: *\nAllow: /\nSitemap: http://localhost:8000/blog/sitemap.xml\n",
  );
});

test("keeps crawlers off the API, downloads and the link handover", () => {
  expect(disallowed).toContain("/api/");
  expect(disallowed).toContain("/download/");
  expect(disallowed).toContain("/link");
});

test("keeps crawlers off the pages that need an account or carry a token", () => {
  for (const path of [
    "/settings",
    "/upload",
    "/sign-in",
    "/sign-up",
    "/verify-email",
    "/forgot-password",
    "/reset-password",
  ]) {
    expect(disallowed).toContain(path);
  }
});

test("leaves work pages crawlable, because an unlisted one answers with noindex", () => {
  expect(rules.allow).toBe("/");
  for (const path of disallowed) {
    expect("/a/".startsWith(path)).toBe(false);
  }
});

test("leaves media crawlable, so a link preview can fetch its image", () => {
  for (const path of disallowed) {
    expect("/media/".startsWith(path)).toBe(false);
  }
});
