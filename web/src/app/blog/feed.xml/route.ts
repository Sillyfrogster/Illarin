import { fetchPostArchive } from "@/lib/api/query";
import { publicationFeed } from "@/lib/publication-feed";

export const dynamic = "force-dynamic";

export async function GET(): Promise<Response> {
  const archive = await fetchPostArchive({});
  return new Response(publicationFeed(archive?.posts ?? []), {
    headers: {
      "cache-control": "public, max-age=300",
      "content-type": "application/rss+xml; charset=utf-8",
    },
  });
}
