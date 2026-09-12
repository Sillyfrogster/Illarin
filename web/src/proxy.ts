import { type NextRequest, NextResponse } from "next/server";
import { routeRequest } from "@/lib/origin-routing";
import {
  fetchWithdrawnPost,
  WITHDRAWN_HEADER,
  WITHDRAWN_ROUTE,
} from "@/lib/publication-withdrawal";

export const config = { matcher: ["/((?!_next/).*)"] };

/** Serves the blog origin's root from the blog tree and keeps the main site's old blog addresses pointing there. */
export async function proxy(request: NextRequest) {
  const route = await routeRequest(
    {
      host: request.headers.get("host"),
      pathname: request.nextUrl.pathname,
      search: request.nextUrl.search,
    },
    fetchWithdrawnPost,
  );
  switch (route.kind) {
    case "pass":
      return;
    case "redirect":
      return NextResponse.redirect(route.to, 308);
    case "rewrite":
      return NextResponse.rewrite(new URL(route.to, request.url), {
        request: { headers: readerHeaders(request) },
      });
    case "withdrawn": {
      const headers = readerHeaders(request);
      headers.set(WITHDRAWN_HEADER, route.slug);
      return NextResponse.rewrite(new URL(WITHDRAWN_ROUTE, request.url), {
        request: { headers },
        status: 410,
      });
    }
  }
}

/** The blog origin reads only, so no page beneath it ever sees a cookie or a header the proxy itself sets. */
function readerHeaders(request: NextRequest): Headers {
  const headers = new Headers(request.headers);
  headers.delete("cookie");
  headers.delete(WITHDRAWN_HEADER);
  return headers;
}
