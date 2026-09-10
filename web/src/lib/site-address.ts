import { siteUrl } from "./site-metadata";

export function siteAddress(path: string): string {
  return new URL(path, siteUrl).href;
}
