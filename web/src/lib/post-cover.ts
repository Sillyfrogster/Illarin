import type { PublicPost } from "@/lib/api/query";

export type PostCover = {
  url: string;
  alt: string;
  width: number;
  height: number;
};

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
