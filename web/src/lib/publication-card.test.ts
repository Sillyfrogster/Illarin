import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";
import {
  type CardSubject,
  cardTitleSize,
  markImage,
  PostCard,
} from "@/lib/publication-card";
import { CARD_SIZE } from "@/lib/publication-metadata";

const LONGEST_TITLE = "Everything the publication editor stores, "
  .repeat(4)
  .slice(0, 160);

async function faces() {
  const files = [
    ["Bodoni Moda", "BodoniModa-Medium.ttf", 500],
    ["Manrope", "Manrope-Medium.ttf", 500],
    ["Manrope", "Manrope-Bold.ttf", 700],
  ] as const;
  return Promise.all(
    files.map(async ([name, file, weight]) => ({
      name,
      data: await readFile(join(process.cwd(), "assets/fonts", file)),
      weight,
      style: "normal" as const,
    })),
  );
}

async function drawn(subject: CardSubject): Promise<Uint8Array> {
  const card = new ImageResponse(PostCard(subject), {
    ...CARD_SIZE,
    fonts: await faces(),
  });
  return new Uint8Array(await card.arrayBuffer());
}

function sizeOf(png: Uint8Array): { width: number; height: number } {
  const header = new DataView(png.buffer, png.byteOffset, png.byteLength);
  return { width: header.getUint32(16), height: header.getUint32(20) };
}

test("a long title steps down to a size that still fits", () => {
  expect(cardTitleSize("Illarin keeps its own writing")).toBe(74);
  expect(cardTitleSize(LONGEST_TITLE)).toBe(46);
});

test("the mark is drawn without a stylesheet", () => {
  const drawing = decodeURIComponent(markImage("#f5f5f2"));
  expect(drawing).toStartWith("data:image/svg+xml;utf8,<svg");
  expect(drawing).toContain('fill="#f5f5f2"');
});

test("a card is the size every link preview expects", async () => {
  const png = await drawn({
    title: "Illarin keeps its own writing",
    app: null,
    plate: null,
  });
  expect(sizeOf(png)).toEqual({ width: 1200, height: 630 });
});

test("the longest title Illarin accepts still makes a whole card", async () => {
  const png = await drawn({
    title: LONGEST_TITLE,
    app: { name: "Lumiverse", mark: null },
    plate: null,
  });
  expect(sizeOf(png)).toEqual({ width: 1200, height: 630 });
});
