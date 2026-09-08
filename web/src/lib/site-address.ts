import { siteUrl } from "./site-metadata";

/** The whole address of a page on Illarin, because the blog answers on its own origin. */
export function siteAddress(path: string): string {
  return new URL(path, siteUrl).href;
}
