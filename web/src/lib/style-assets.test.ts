import { expect, test } from "bun:test";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, resolve } from "node:path";

const ROOT = join(import.meta.dir, "../..");

function stylesheets(directory: string): string[] {
  const found: string[] = [];
  for (const entry of readdirSync(join(ROOT, directory))) {
    const path = join(directory, entry);
    if (statSync(join(ROOT, path)).isDirectory()) {
      found.push(...stylesheets(path));
      continue;
    }
    if (path.endsWith(".css")) found.push(path);
  }
  return found;
}

function references(source: string): string[] {
  return [...source.matchAll(/url\("([^"]+)"\)/g)]
    .map((found) => found[1])
    .filter((one) => !/^(https?:|data:|\/)/.test(one));
}

test("every picture a stylesheet reaches for is where it says it is", () => {
  const missing: string[] = [];
  for (const sheet of stylesheets("src")) {
    const here = resolve(join(ROOT, sheet), "..");
    for (const reference of references(
      readFileSync(join(ROOT, sheet), "utf8"),
    )) {
      try {
        statSync(resolve(here, reference));
      } catch {
        missing.push(`${sheet} -> ${reference}`);
      }
    }
  }
  expect(missing).toEqual([]);
});
