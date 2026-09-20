import { blogFeedResponse } from "@/lib/blog-feed-route";

export const dynamic = "force-dynamic";

export function GET(): Promise<Response> {
  return blogFeedResponse("rss");
}
