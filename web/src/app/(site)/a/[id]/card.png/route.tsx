import { notFound } from "next/navigation";
import { fetchWork } from "@/lib/api/query";
import { renderLinkCard } from "@/lib/link-card";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";
import { isWorkId } from "@/lib/work-url";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
): Promise<Response> {
  const { id } = await params;
  if (!isWorkId(id)) notFound();
  const work = await fetchWork(id);
  if (!work || work.lifecycle !== "published" || work.takedown) notFound();
  const cover = work.media.find((image) => image.isCover);
  return renderLinkCard({
    title: workDisplayName(work.name),
    eyebrow: TYPE_LABELS[work.type],
    image: cover
      ? { url: cover.detailUrl, width: cover.width, height: cover.height }
      : null,
    byline: `By @${work.creator}`,
    description: work.blurb,
  });
}
