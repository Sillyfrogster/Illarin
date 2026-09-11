import type { PostByline } from "@/lib/api/query";
import { addressOn } from "./address";
import { bylineProfilePath } from "./byline";
import { profilePath } from "./profile-address";
import { siteUrl } from "./site-metadata";

export function siteAddress(path: string): string {
  return addressOn(siteUrl, path);
}

export function profileAddress(handle: string): string {
  return siteAddress(profilePath(handle));
}

export function bylineProfile(byline: PostByline): string | null {
  const path = bylineProfilePath(byline);
  return path ? siteAddress(path) : null;
}
