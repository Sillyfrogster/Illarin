import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import {
  importsOf,
  reachableFrom,
  SOURCE_ROOT,
  sourceFiles,
} from "./source-graph";

const READER_FILES = [
  "src/app/blog/[slug]/page.tsx",
  "src/components/blog/Article.tsx",
  "src/components/blog/ArticleIdentity.tsx",
  "src/components/blog/ArticleBody.tsx",
  "src/components/blog/ArticleContents.tsx",
  "src/components/blog/ShareArticle.tsx",
  "src/app/blog/feed.xml/route.ts",
];

test("no editor code reaches the published post page", () => {
  for (const entry of READER_FILES) {
    for (const path of reachableFrom(entry)) {
      expect(path).not.toStartWith("@tiptap");
      expect(path).not.toContain("prosemirror");
      expect(path).not.toContain("blog/writing");
    }
  }
});

test("the editor is the only place Tiptap is imported", () => {
  for (const path of sourceFiles()) {
    const found = importsOf(readFileSync(join(SOURCE_ROOT, path), "utf8"));
    if (found.some((one) => one.startsWith("@tiptap"))) {
      expect(path).toContain("blog/writing");
    }
  }
});
