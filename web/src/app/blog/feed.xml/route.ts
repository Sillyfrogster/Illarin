import { publicationFeedResponse } from "@/lib/publication-feed-route";

export const dynamic = "force-dynamic";

export function GET(): Promise<Response> {
  return publicationFeedResponse("rss");
}
