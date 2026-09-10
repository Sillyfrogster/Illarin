import { blogUrl, siteUrl } from "./site-metadata";

export function blogAddress(path: string): string {
  return new URL(path, blogUrl).href;
}

export function isSafeAddress(href: string): boolean {
  const address = href.trim();
  for (const letter of address) {
    const code = letter.codePointAt(0) ?? 0;
    if (code <= 0x20 || code === 0x7f) return false;
  }
  return (
    (address.startsWith("https://") && address.length > "https://".length) ||
    (address.startsWith("mailto:") && address.length > "mailto:".length)
  );
}

export function leavesIllarin(href: string): boolean {
  if (!href.startsWith("https://")) return false;
  try {
    const going = new URL(href).hostname.toLowerCase();
    const here = new URL(siteUrl).hostname.toLowerCase();
    return going !== here && !going.endsWith(`.${here}`);
  } catch {
    return true;
  }
}

export function normalizedSlug(candidate: string): string {
  return candidate
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80)
    .replace(/-+$/, "");
}

export function postPermalink(slug: string): string {
  return blogAddress(`/blog/${encodeURI(slug)}`);
}
