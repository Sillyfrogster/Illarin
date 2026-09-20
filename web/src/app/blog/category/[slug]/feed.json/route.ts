import { categoryFeedResponse } from "@/lib/blog-feed-route";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ slug: string }> },
): Promise<Response> {
  return categoryFeedResponse((await params).slug, "json");
}
