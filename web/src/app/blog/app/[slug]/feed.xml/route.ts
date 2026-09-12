import { scopedFeedResponse } from "@/lib/publication-feed-route";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ slug: string }> },
): Promise<Response> {
  return scopedFeedResponse("app", (await params).slug, "rss");
}
