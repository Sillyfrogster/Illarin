import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { notFound } from "next/navigation";
import { ImageResponse } from "next/og";
import { fetchPublishedPost, type PublicPost } from "@/lib/api/query";
import { type CardSubject, PostCard } from "@/lib/publication-card";
import { CARD_SIZE, ILLARIN_APP } from "@/lib/publication-metadata";
import { mediaUrl } from "@/lib/site-metadata";

export const dynamic = "force-dynamic";

const CARD_MAX_AGE = 86400;

const FONTS = [
  { name: "Outfit", file: "Outfit-SemiBold.ttf", weight: 600 },
  { name: "DM Sans", file: "DMSans-Medium.ttf", weight: 500 },
] as const;

const resources = Promise.all([
  faces(),
  readFile(
    join(process.cwd(), "public/brand/illarin-horizontal-white.svg"),
    "utf8",
  ).then((svg) => `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`),
]);

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ slug: string }> },
): Promise<Response> {
  const post = await fetchPublishedPost(
    decodeURIComponent((await params).slug),
  );
  if (!post) notFound();
  const [subject, [fonts, logo]] = await Promise.all([
    composed(post),
    resources,
  ]);
  return new ImageResponse(PostCard({ ...subject, logo }), {
    ...CARD_SIZE,
    fonts,
    headers: {
      "cache-control": `public, max-age=${CARD_MAX_AGE}`,
      "content-type": "image/png",
    },
  });
}

async function composed(post: PublicPost): Promise<CardSubject> {
  const header = post.header
    ? post.media.find((one) => one.id === post.header?.mediaId)
    : undefined;
  const app = post.release?.app ?? post.byline.app ?? null;
  const named = app && app.slug !== ILLARIN_APP ? app : null;
  return {
    title: post.title,
    app: named
      ? {
          name: named.name,
          mark: named.mark ? await drawable(named.mark.url) : null,
        }
      : null,
    plate: header ? await drawable(header.url) : null,
  };
}

async function drawable(address: string): Promise<string | null> {
  try {
    const response = await fetch(new URL(address, mediaUrl));
    const type = response.headers.get("content-type") ?? "";
    if (!response.ok || !type.startsWith("image/")) return null;
    const bytes = Buffer.from(await response.arrayBuffer());
    return `data:${type};base64,${bytes.toString("base64")}`;
  } catch {
    return null;
  }
}

async function faces() {
  return Promise.all(
    FONTS.map(async (face) => ({
      name: face.name,
      data: await readFile(join(process.cwd(), "assets/fonts", face.file)),
      weight: face.weight,
      style: "normal" as const,
    })),
  );
}
