import { type NextRequest, NextResponse } from "next/server";
import {
  fetchWithdrawnPost,
  movedTo,
  postAddressIn,
  WITHDRAWN_HEADER,
  WITHDRAWN_ROUTE,
} from "@/lib/publication-withdrawal";

export const config = { matcher: "/blog/:slug" };

/** Answers a withdrawn address with a tombstone, which needs a status a page cannot set. */
export async function proxy(request: NextRequest) {
  const asked = postAddressIn(request.nextUrl.pathname);
  if (!asked) return;
  const withdrawn = await fetchWithdrawnPost(asked);
  if (!withdrawn) return;
  const moved = movedTo(asked, withdrawn);
  if (moved) return NextResponse.redirect(new URL(moved, request.url), 308);
  const headers = new Headers(request.headers);
  headers.set(WITHDRAWN_HEADER, withdrawn.slug);
  return NextResponse.rewrite(new URL(WITHDRAWN_ROUTE, request.url), {
    request: { headers },
    status: 410,
  });
}
