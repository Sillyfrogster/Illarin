import { notFound } from "next/navigation";
import { fetchPublishedPost } from "@/lib/api/query";
import { bylineName } from "@/lib/byline";
import { renderLinkCard } from "@/lib/link-card";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ slug: string }> },
): Promise<Response> {
  const post = await fetchPublishedPost(
    decodeURIComponent((await params).slug),
  );
  if (!post) notFound();
  const header = post.media.find((image) => image.id === post.header?.mediaId);
  return renderLinkCard({
    title: post.title,
    eyebrow: "Blog",
    image: post.linkCardImage ?? header,
    byline: `By ${bylineName(post.byline)}`,
    description: post.summary,
    footer: new Date(post.publishedAt).toLocaleDateString("en-US", {
      month: "long",
      day: "numeric",
      year: "numeric",
      timeZone: "UTC",
    }),
  });
}
