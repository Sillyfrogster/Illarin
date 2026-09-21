import { type NextRequest, NextResponse } from "next/server";
import { routeRequest } from "@/lib/origin-routing";
import {
  fetchUnpublishedPost,
  UNPUBLISHED_HEADER,
  UNPUBLISHED_ROUTE,
} from "@/lib/post-unpublishing";

export const config = { matcher: ["/((?!_next/).*)"] };

export async function proxy(request: NextRequest) {
  const route = await routeRequest(
    { pathname: request.nextUrl.pathname },
    fetchUnpublishedPost,
  );
  switch (route.kind) {
    case "pass":
      return;
    case "redirect":
      return NextResponse.redirect(route.to, 308);
    case "unpublished": {
      const headers = new Headers(request.headers);
      headers.set(UNPUBLISHED_HEADER, route.slug);
      return NextResponse.rewrite(new URL(UNPUBLISHED_ROUTE, request.url), {
        request: { headers },
        status: 410,
      });
    }
  }
}
