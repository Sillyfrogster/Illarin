import { addressOn } from "./address";
import { postPath } from "./blog-paths";
import { siteUrl } from "./site-metadata";

export function blogAddress(path: string, search = ""): string {
  return addressOn(siteUrl, path, search);
}

export function postPermalink(slug: string): string {
  return blogAddress(postPath(slug));
}
