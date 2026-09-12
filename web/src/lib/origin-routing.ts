import { blogAddress, blogHostname, postPermalink } from "@/lib/blog-address";
import {
  BLOG_TREE,
  postAddressIn,
  type WithdrawnPost,
} from "@/lib/publication-withdrawal";

export type Route =
  | { kind: "pass" }
  | { kind: "rewrite"; to: string }
  | { kind: "redirect"; to: string }
  | { kind: "withdrawn"; slug: string };

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
  withdrawnPost: (slug: string) => Promise<WithdrawnPost | null>,
): Promise<Route> {
  if (hostnameOf(asked.host) === blogHostname)
    return onBlogOrigin(asked, withdrawnPost);
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
  withdrawnPost: (slug: string) => Promise<WithdrawnPost | null>,
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
  const withdrawn = slug ? await withdrawnPost(slug) : null;
  if (withdrawn && slug !== withdrawn.slug) {
    return { kind: "redirect", to: postPermalink(withdrawn.slug) };
  }
  if (withdrawn) return { kind: "withdrawn", slug: withdrawn.slug };
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
