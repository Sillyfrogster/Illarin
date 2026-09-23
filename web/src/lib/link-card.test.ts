import { afterAll, afterEach, expect, spyOn, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { GET as profileCard } from "@/app/(site)/[profile]/card.png/route";
import { GET as workCard } from "@/app/(site)/a/[id]/card.png/route";
import { GET as postCard } from "@/app/blog/[slug]/card.png/route";

const id = "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a";
const request = new Request("http://localhost/card.png?nsfw=shown", {
  headers: { cookie: "session=owner" },
});
const work = {
  name: "The archivist",
  type: "character",
  creator: "midnight.archive",
  blurb: "Keeps a record of every visitor.",
  lifecycle: "published",
  isNsfw: true,
  media: [
    {
      isCover: true,
      detailUrl: "/media/cover/detail_blurred/2",
      width: 600,
      height: 900,
    },
  ],
};
const image = await readFile("public/site-card.png");
const originalFetch = globalThis.fetch;
const fetchSpy = spyOn(globalThis, "fetch");
afterEach(() => fetchSpy.mockReset());
afterAll(() => fetchSpy.mockRestore());

function publicData(data: unknown) {
  const paths: string[] = [];
  const serve: typeof fetch = Object.assign(
    async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input));
      if (!["http:", "https:"].includes(url.protocol))
        return originalFetch(input, init);
      const path = url.pathname;
      paths.push(path);
      expect(new Headers(init?.headers).has("cookie")).toBe(false);
      return path.startsWith("/media/")
        ? new Response(image, { headers: { "content-type": "image/png" } })
        : Response.json(data);
    },
    { preconnect: originalFetch.preconnect },
  );
  fetchSpy.mockImplementation(serve);
  return paths;
}

async function expectCard(response: Response) {
  expect(response.headers.get("cache-control")).toBe("public, max-age=300");
  expect(response.headers.get("content-type")).toBe("image/png");
  const bytes = Buffer.from(await response.arrayBuffer());
  expect(bytes.subarray(1, 4).toString()).toBe("PNG");
  expect([bytes.readUInt32BE(16), bytes.readUInt32BE(20)]).toEqual([1200, 630]);
  return bytes;
}

test("a work card reads public blurred media even with an owner's cookie and shown preference", async () => {
  const paths = publicData(work);
  await expectCard(
    await workCard(request, { params: Promise.resolve({ id }) }),
  );
  expect(paths).toEqual([`/v1/works/${id}`, "/media/cover/detail_blurred/2"]);
});

test("drafts and taken-down works refuse a card before reading artwork", async () => {
  for (const refused of [
    { ...work, lifecycle: "draft" },
    { ...work, takedown: { reason: "Removed" } },
    null,
  ]) {
    const paths = publicData(refused);
    await expect(
      workCard(request, { params: Promise.resolve({ id }) }),
    ).rejects.toThrow("NEXT_HTTP_ERROR_FALLBACK;404");
    expect(paths).toEqual([`/v1/works/${id}`]);
  }
});

test("profile cards render with a banner, an avatar alone, or initials and fallback art", async () => {
  for (const pictures of [
    {
      banner: { url: "/media/banner/detail/2", width: 1200, height: 630 },
      avatar: { url: "/media/avatar/detail/2", width: 600, height: 600 },
    },
    { avatar: { url: "/media/avatar/detail/2", width: 600, height: 600 } },
    {},
  ]) {
    const paths = publicData({
      ...pictures,
      displayName: "Midnight Vale Storytelling Collective ".repeat(8),
      handle: "midnight.storytelling.collective",
      biography: "AnUnbrokenDescription".repeat(30),
      works: 24,
      followers: 386,
    });
    await expectCard(
      await profileCard(request, {
        params: Promise.resolve({
          profile: "@midnight.storytelling.collective",
        }),
      }),
    );
    expect(paths.filter((path) => path.startsWith("/media/"))).toHaveLength(
      Object.keys(pictures).length,
    );
  }
});

test("a post composes its OG artwork before the header, and needs no stats", async () => {
  const post = {
    title: "A title that needs several lines ".repeat(12),
    summary: "LongUnbrokenSummary".repeat(40),
    byline: { displayName: "A long author name ".repeat(20), handle: "wren" },
    publishedAt: "2026-09-22T00:00:00Z",
    header: { mediaId: "header" },
    media: [
      { id: "header", url: "/media/header/detail/2", width: 1200, height: 630 },
    ],
  };
  for (const linkCardImage of [
    { url: "/media/override/og/2", width: 1200, height: 630 },
    undefined,
  ]) {
    const paths = publicData({ ...post, linkCardImage });
    await expectCard(
      await postCard(request, {
        params: Promise.resolve({ slug: "first-post" }),
      }),
    );
    expect(paths).toEqual([
      "/v1/posts/first-post",
      linkCardImage?.url ?? "/media/header/detail/2",
    ]);
  }
});

test("text past the visible lines cannot move the profile details or footer", async () => {
  const cards: Buffer[] = [];
  for (const repeats of [3, 12]) {
    publicData({
      displayName: "MidnightValeStorytellingCollective".repeat(repeats),
      biography: "AnUnbrokenDescription".repeat(repeats * 3),
      handle: "midnight.collective",
      works: 24,
      followers: 386,
    });
    cards.push(
      await expectCard(
        await profileCard(request, {
          params: Promise.resolve({ profile: "@midnight.collective" }),
        }),
      ),
    );
  }
  expect(cards[0].equals(cards[1])).toBe(true);
});
