import type { PostBody } from "@/lib/post-body";

const CONTENTS_FLOOR = 3;

const CONTENTS_DEPTH = 3;

export type PostContentsEntry = {
  anchor: string;
  label: string;
  level: number;
};

export function postContents(body: PostBody): PostContentsEntry[] {
  const entries: PostContentsEntry[] = [];
  for (const block of body.content) {
    if (block.type !== "heading") continue;
    if (block.level > CONTENTS_DEPTH) continue;
    if (!block.anchor) continue;
    const label = block.content
      .map((span) => span.text)
      .join("")
      .trim();
    if (label === "") continue;
    entries.push({ anchor: block.anchor, label, level: block.level });
  }
  return entries.length >= CONTENTS_FLOOR ? entries : [];
}
