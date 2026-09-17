import { api } from "@/lib/api/client";

import type { PublicPost, WithdrawnPost } from "@/lib/api/shapes";
export type { WithdrawnPost };

export const WITHDRAWAL_MESSAGE = "Illarin took this post out of public view.";

export const WITHDRAWN_HEADER = "x-withdrawn-post";

/** Where the blog's route tree lives inside the application. The proxy maps the blog origin's root onto it. */
export const BLOG_TREE = "/blog";

export const WITHDRAWN_ROUTE = `${BLOG_TREE}/withdrawn`;

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

export async function fetchWithdrawnPost(
  slug: string,
): Promise<WithdrawnPost | null> {
  const answer = await api<PublicPost>(
    "GET",
    `/v1/posts/${encodeURIComponent(slug)}`,
  ).catch(() => null);
  if (!answer || answer.response.status !== 410) return null;
  const withdrawn = answer.error as WithdrawnPost | undefined;
  return withdrawn?.slug ? withdrawn : null;
}
