import { blogUrl, siteUrl } from "./site-metadata";

/** The whole address of anything the blog publishes, on the blog's own origin. */
export function blogAddress(path: string): string {
  return new URL(path, blogUrl).href;
}

/** The only schemes a post link may use. Go refuses every other one. */
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

/** Whether a link in a post leaves Illarin, which decides how it opens. */
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

/** The address form Go stores, so an editor can preview it before saving. */
export function normalizedSlug(candidate: string): string {
  return candidate
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80)
    .replace(/-+$/, "");
}

/** The whole address a post answers to, which is what sharing hands to someone else. */
export function postPermalink(slug: string): string {
  return blogAddress(`/blog/${encodeURI(slug)}`);
}
