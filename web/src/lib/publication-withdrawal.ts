import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

export type WithdrawnPost = components["schemas"]["WithdrawnPost"];

/** What every withdrawn address says, whatever was written privately about why. */
export const WITHDRAWAL_MESSAGE = "Illarin took this post out of public view.";

/** The header the tombstone is reached through, so its own address answers nothing. */
export const WITHDRAWN_HEADER = "x-withdrawn-post";

/** Where the tombstone is rendered from, which a reader never sees in the address bar. */
export const WITHDRAWN_ROUTE = "/blog/withdrawn";

/** Everything the publication puts a post at lives under this. */
export const POST_ADDRESS = "/blog/";

/** The address a path asks a post for, and nothing for a feed, a sitemap or an archive. */
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

/** The address a withdrawn post's tombstone lives at, when the reader asked at another one. */
export function movedTo(
  asked: string,
  withdrawn: WithdrawnPost,
): string | null {
  if (asked === withdrawn.slug) return null;
  return `${POST_ADDRESS}${encodeURI(withdrawn.slug)}`;
}

/** The tombstone behind one address, current or former, or nothing if it has none. */
export async function fetchWithdrawnPost(
  slug: string,
): Promise<WithdrawnPost | null> {
  const { error, response } = await api.GET("/v1/posts/{slug}", {
    params: { path: { slug } },
  });
  if (response.status !== 410) return null;
  const withdrawn = error as WithdrawnPost | undefined;
  return withdrawn?.slug ? withdrawn : null;
}
