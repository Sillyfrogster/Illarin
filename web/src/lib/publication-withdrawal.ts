import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

export type WithdrawnPost = components["schemas"]["WithdrawnPost"];

export const WITHDRAWAL_MESSAGE = "Illarin took this post out of public view.";

export const WITHDRAWN_HEADER = "x-withdrawn-post";

export const WITHDRAWN_ROUTE = "/blog/withdrawn";

export const POST_ADDRESS = "/blog/";

export function postAddressIn(pathname: string): string | null {
  if (!pathname.startsWith(POST_ADDRESS)) return null;
  const asked = pathname.slice(POST_ADDRESS.length);
  if (asked === "" || asked.includes("/") || asked.includes(".")) return null;
  try {
    return decodeURIComponent(asked);
  } catch {
    return null;
  }
}

export function movedTo(
  asked: string,
  withdrawn: WithdrawnPost,
): string | null {
  if (asked === withdrawn.slug) return null;
  return `${POST_ADDRESS}${encodeURI(withdrawn.slug)}`;
}

export async function fetchWithdrawnPost(
  slug: string,
): Promise<WithdrawnPost | null> {
  const answer = await api
    .GET("/v1/posts/{slug}", { params: { path: { slug } } })
    .catch(() => null);
  if (!answer || answer.response.status !== 410) return null;
  const withdrawn = answer.error as WithdrawnPost | undefined;
  return withdrawn?.slug ? withdrawn : null;
}
