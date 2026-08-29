import { siteUrl } from "./site-metadata";

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
