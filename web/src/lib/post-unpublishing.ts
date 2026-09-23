import { api } from "@/lib/api/client";

import type { PublicPost, UnpublishedPost } from "@/lib/api/shapes";
export type { UnpublishedPost };

export const UNPUBLISHED_MESSAGE = "Illarin took this post out of public view.";

export const UNPUBLISHED_HEADER = "x-unpublished-post";

export const BLOG_TREE = "/blog";

export const UNPUBLISHED_ROUTE = `${BLOG_TREE}/unpublished`;

/** The slug a path under the blog tree asks for, when it asks for one post and nothing beneath it. */
export function postAddressIn(pathname: string): string | null {
  if (!pathname.startsWith(`${BLOG_TREE}/`)) return null;
  const asked = pathname.slice(BLOG_TREE.length + 1);
  if (asked === "" || asked.includes("/") || asked.includes(".")) return null;
  try {
    return decodeURIComponent(asked);
  } catch {
    return null;
  }
}

export async function fetchUnpublishedPost(
  slug: string,
): Promise<UnpublishedPost | null> {
  const answer = await api<PublicPost>(
    "GET",
    `/v1/posts/${encodeURIComponent(slug)}`,
  ).catch(() => null);
  if (!answer || answer.response.status !== 410) return null;
  const unpublished = answer.error as UnpublishedPost | undefined;
  return unpublished?.slug ? unpublished : null;
}
