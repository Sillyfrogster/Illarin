import type { PostDocument } from "@/lib/post-document";

/** How many entries a post needs before its structure is worth listing. */
const CONTENTS_FLOOR = 3;

/** How deep the contents follow a post's structure. */
const CONTENTS_DEPTH = 3;

export type PostContentsEntry = {
  anchor: string;
  label: string;
  level: number;
};

/** The outline a post's own headings make, or nothing where they make too little. */
export function postContents(document: PostDocument): PostContentsEntry[] {
  const entries: PostContentsEntry[] = [];
  for (const block of document.content) {
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
