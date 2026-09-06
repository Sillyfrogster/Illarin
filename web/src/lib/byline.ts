import type { PostByline } from "@/lib/api/query";
import { profileAddress } from "@/lib/profile-address";

/** The name a byline is read under, falling back to the handle it was captured with. */
export function bylineName(byline: PostByline): string {
  return byline.displayName || `@${byline.handle}`;
}

/** Where a byline sends a reader, which is nowhere when no account stands behind it. */
export function bylineProfile(byline: PostByline): string | null {
  return byline.historical ? null : profileAddress(byline.handle);
}
