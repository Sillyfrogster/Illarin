import { expect, test } from "bun:test";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

/** Everything the signed-out reader downloads for one published post. */
const READER_FILES = [
  "src/app/blog/[slug]/page.tsx",
  "src/components/publication/Article.tsx",
  "src/components/publication/ArticleIdentity.tsx",
  "src/components/publication/PostBody.tsx",
  "src/components/publication/ArticleContents.tsx",
  "src/components/publication/ShareArticle.tsx",
  "src/app/blog/feed.xml/route.ts",
];

const ROOT = join(import.meta.dir, "../..");

function imports(source: string): string[] {
  return [...source.matchAll(/from\s+"([^"]+)"/g)].map((found) => found[1]);
}

function reachable(entry: string, seen = new Set<string>()): string[] {
  if (seen.has(entry)) return [];
  seen.add(entry);
  const source = readFileSync(join(ROOT, entry), "utf8");
  const found = imports(source);
  for (const path of found) {
    if (!path.startsWith("@/")) continue;
    const resolved = resolve(path.slice(2));
    if (resolved) found.push(...reachable(resolved, seen));
  }
  return found;
}

function resolve(relative: string): string | null {
  for (const suffix of [".tsx", ".ts", "/index.tsx", "/index.ts"]) {
    const candidate = join("src", relative + suffix);
    try {
      statSync(join(ROOT, candidate));
      return candidate;
    } catch {}
  }
  return null;
}

test("no editor code reaches the published post page", () => {
  for (const entry of READER_FILES) {
    for (const path of reachable(entry)) {
      expect(path).not.toStartWith("@tiptap");
      expect(path).not.toContain("prosemirror");
      expect(path).not.toContain("publication/editor");
    }
  }
});

test("the editor is the only place Tiptap is imported", () => {
  const wanting: string[] = [];
  const walk = (directory: string) => {
    for (const entry of readdirSync(join(ROOT, directory))) {
      const path = join(directory, entry);
      if (statSync(join(ROOT, path)).isDirectory()) {
        walk(path);
        continue;
      }
      if (!path.endsWith(".ts") && !path.endsWith(".tsx")) continue;
      const found = imports(readFileSync(join(ROOT, path), "utf8"));
      if (found.some((one) => one.startsWith("@tiptap"))) wanting.push(path);
    }
  };
  walk("src");
  for (const path of wanting) {
    expect(path).toContain("publication/editor");
  }
});
