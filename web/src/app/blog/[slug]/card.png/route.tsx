import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { notFound } from "next/navigation";
import { ImageResponse } from "next/og";
import { fetchPublishedPost, type PublicPost } from "@/lib/api/query";
import { type CardSubject, PostCard } from "@/lib/publication-card";
import { CARD_SIZE, ILLARIN_APP } from "@/lib/publication-metadata";
import { mediaUrl } from "@/lib/site-metadata";

export const dynamic = "force-dynamic";

/** How long a generated card may be kept before it is composed again. */
const CARD_MAX_AGE = 86400;

const FONTS = [
  { name: "Bodoni Moda", file: "BodoniModa-Medium.ttf", weight: 500 },
  { name: "Manrope", file: "Manrope-Medium.ttf", weight: 500 },
  { name: "Manrope", file: "Manrope-Bold.ttf", weight: 700 },
] as const;

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ slug: string }> },
): Promise<Response> {
  const post = await fetchPublishedPost(
    decodeURIComponent((await params).slug),
  );
  if (!post) notFound();
  const [subject, fonts] = await Promise.all([composed(post), faces()]);
  return new ImageResponse(PostCard(subject), {
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

/** Reads uploaded bytes into the card, leaving the picture out when they cannot be had. */
async function drawable(address: string): Promise<string | null> {
  try {
    const response = await fetch(new URL(address, mediaUrl));
    if (!response.ok) return null;
    const type = response.headers.get("content-type") ?? "image/png";
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
