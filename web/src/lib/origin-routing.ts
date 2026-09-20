import { blogAddress, blogHostname, postPermalink } from "@/lib/blog-address";
import {
  BLOG_TREE,
  postAddressIn,
  type UnpublishedPost,
} from "@/lib/post-unpublishing";

export type Route =
  | { kind: "pass" }
  | { kind: "rewrite"; to: string }
  | { kind: "redirect"; to: string }
  | { kind: "unpublished"; slug: string };

export type Asked = { host: string | null; pathname: string; search: string };

/** Files the blog origin serves from the application root rather than from the blog tree. */
const ROOT_FILES = new Set([
  "/favicon.ico",
  "/icon.svg",
  "/apple-icon.png",
  "/site-card.png",
  "/site.webmanifest",
]);

/** Decides what one request gets, from the hostname it arrived on and the path it asked for. */
export async function routeRequest(
  asked: Asked,
  unpublishedPost: (slug: string) => Promise<UnpublishedPost | null>,
): Promise<Route> {
  if (hostnameOf(asked.host) === blogHostname)
    return onBlogOrigin(asked, unpublishedPost);
  if (
    asked.pathname === BLOG_TREE ||
    asked.pathname.startsWith(`${BLOG_TREE}/`)
  ) {
    const beneath = asked.pathname.slice(BLOG_TREE.length) || "/";
    return { kind: "redirect", to: blogAddress(beneath, asked.search) };
  }
  return { kind: "pass" };
}

async function onBlogOrigin(
  asked: Asked,
  unpublishedPost: (slug: string) => Promise<UnpublishedPost | null>,
): Promise<Route> {
  if (
    asked.pathname.startsWith("/_next/") ||
    asked.pathname.startsWith("/brand/") ||
    ROOT_FILES.has(asked.pathname)
  ) {
    return { kind: "pass" };
  }
  const inside =
    asked.pathname === "/" ? BLOG_TREE : `${BLOG_TREE}${asked.pathname}`;
  const slug = postAddressIn(inside);
  const unpublished = slug ? await unpublishedPost(slug) : null;
  if (unpublished && slug !== unpublished.slug) {
    return { kind: "redirect", to: postPermalink(unpublished.slug) };
  }
  if (unpublished) return { kind: "unpublished", slug: unpublished.slug };
  return { kind: "rewrite", to: inside + asked.search };
}

function hostnameOf(host: string | null): string {
  if (!host) return "";
  try {
    return new URL(`http://${host}`).hostname.toLowerCase();
  } catch {
    return "";
  }
}
