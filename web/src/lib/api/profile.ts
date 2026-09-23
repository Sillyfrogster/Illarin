import { type Answer, ask } from "./request";
import type { CreatorFollow, Profile, SaveProfileRequest } from "./shapes";

export type ProfilePictureKind = "avatar" | "banner";

/** Saves the words on the owner's profile and answers with the whole profile. */
export function saveProfile(
  draft: SaveProfileRequest,
): Promise<Answer<Profile>> {
  return ask<Profile>("PUT", "/account/profile", { body: draft });
}

/** Uploads a new avatar or banner and answers with the updated profile */
export function saveProfilePicture(
  kind: ProfilePictureKind,
  file: File,
): Promise<Answer<Profile>> {
  const body = new FormData();
  body.append("file", file);
  return ask<Profile>("PUT", `/account/profile/${kind}`, { body });
}

export function removeProfilePicture(
  kind: ProfilePictureKind,
): Promise<Answer<Profile>> {
  return ask<Profile>("DELETE", `/account/profile/${kind}`);
}

/** Records which of the owner's works show first, in this order. */
export function saveFeaturedWorks(workIds: string[]): Promise<Answer<Profile>> {
  return ask<Profile>("PUT", "/account/profile/featured", {
    body: { workIds },
  });
}

export function followCreator(handle: string): Promise<Answer<CreatorFollow>> {
  return ask<CreatorFollow>(
    "PUT",
    `/profiles/${encodeURIComponent(handle)}/follow`,
  );
}

export function stopFollowingCreator(
  handle: string,
): Promise<Answer<CreatorFollow>> {
  return ask<CreatorFollow>(
    "DELETE",
    `/profiles/${encodeURIComponent(handle)}/follow`,
  );
}
