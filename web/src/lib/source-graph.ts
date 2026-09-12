import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";

export const SOURCE_ROOT = join(import.meta.dir, "../..");

export function importsOf(source: string): string[] {
  return [...source.matchAll(/from\s+"([^"]+)"/g)].map((found) => found[1]);
}

/** Every module an entry file reaches, with package names as written and repository files as paths under src. */
export function reachableFrom(
  entry: string,
  seen = new Set<string>(),
): string[] {
  if (seen.has(entry)) return [];
  seen.add(entry);
  const found: string[] = [];
  for (const path of importsOf(
    readFileSync(join(SOURCE_ROOT, entry), "utf8"),
  )) {
    const resolved = path.startsWith("@/")
      ? resolveSource(path.slice(2))
      : path.startsWith(".")
        ? resolveSource(path, dirname(entry))
        : null;
    if (!resolved) {
      found.push(path);
      continue;
    }
    found.push(resolved, ...reachableFrom(resolved, seen));
  }
  return found;
}

export function resolveSource(relative: string, base = "src"): string | null {
  for (const suffix of [".tsx", ".ts", "/index.tsx", "/index.ts"]) {
    const candidate = join(base, relative + suffix);
    try {
      statSync(join(SOURCE_ROOT, candidate));
      return candidate;
    } catch {}
  }
  return null;
}

export function sourceFiles(directory = "src"): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(join(SOURCE_ROOT, directory))) {
    const path = join(directory, entry);
    if (statSync(join(SOURCE_ROOT, path)).isDirectory()) {
      files.push(...sourceFiles(path));
      continue;
    }
    if (path.endsWith(".ts") || path.endsWith(".tsx")) files.push(path);
  }
  return files;
}

export function isClientModule(path: string): boolean {
  return /^\s*"use client";/.test(
    readFileSync(join(SOURCE_ROOT, path), "utf8"),
  );
}
