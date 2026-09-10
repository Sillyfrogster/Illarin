import type { PostByline } from "@/lib/api/query";
import { profileAddress } from "@/lib/profile-address";

export function bylineName(byline: PostByline): string {
  return byline.displayName || `@${byline.handle}`;
}

export function bylineProfile(byline: PostByline): string | null {
  return byline.historical ? null : profileAddress(byline.handle);
}
