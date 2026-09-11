import type { PostByline } from "@/lib/api/query";
import { profilePath } from "@/lib/profile-address";

export function bylineName(byline: PostByline): string {
  return byline.displayName || `@${byline.handle}`;
}

export function bylineProfilePath(byline: PostByline): string | null {
  return byline.historical ? null : profilePath(byline.handle);
}
