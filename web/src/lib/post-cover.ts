import type { PublicPost } from "@/lib/api/query";

/** The picture a post is shown by where it is not being read, which is its own header. */
export type PostCover = {
  url: string;
  alt: string;
  width: number;
  height: number;
};

/** The header picture a post can be led with. Illarin never substitutes artwork for a missing one. */
export function postCover(post: PublicPost | null): PostCover | null {
  const header = post?.header;
  if (!post || !header) return null;
  const held = post.media.find((one) => one.id === header.mediaId);
  if (!held) return null;
  return {
    url: held.url,
    alt: header.alt,
    width: held.width,
    height: held.height,
  };
}
