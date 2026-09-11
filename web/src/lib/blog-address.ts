import { addressOn } from "./address";
import { postPath } from "./blog-paths";
import { blogUrl } from "./site-metadata";

export const blogHostname = new URL(blogUrl).hostname;

export function blogAddress(path: string, search = ""): string {
  return addressOn(blogUrl, path, search);
}

export function postPermalink(slug: string): string {
  return blogAddress(postPath(slug));
}
