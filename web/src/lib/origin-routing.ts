import { postPermalink } from "@/lib/blog-address";
import { postAddressIn, type UnpublishedPost } from "@/lib/post-unpublishing";

export type Route =
  | { kind: "pass" }
  | { kind: "redirect"; to: string }
  | { kind: "unpublished"; slug: string };

export type Asked = { pathname: string };

export async function routeRequest(
  asked: Asked,
  unpublishedPost: (slug: string) => Promise<UnpublishedPost | null>,
): Promise<Route> {
  const slug = postAddressIn(asked.pathname);
  if (!slug) return { kind: "pass" };
  const unpublished = slug ? await unpublishedPost(slug) : null;
  if (unpublished && slug !== unpublished.slug) {
    return { kind: "redirect", to: postPermalink(unpublished.slug) };
  }
  if (unpublished) return { kind: "unpublished", slug: unpublished.slug };
  return { kind: "pass" };
}
