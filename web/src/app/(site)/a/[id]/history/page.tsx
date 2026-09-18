import { cookies } from "next/headers";
import { notFound, permanentRedirect } from "next/navigation";
import { fetchWork } from "@/lib/api/query";
import { isWorkId, workHistoryHref } from "@/lib/work-url";

/** Sends the old history address to the work page, which keeps any version fragment. */
export default async function WorkHistoryPage({
  params,
}: PageProps<"/a/[id]/history">) {
  const { id } = await params;
  if (!isWorkId(id)) notFound();
  const work = await fetchWork(id, (await cookies()).toString());
  if (!work) notFound();
  permanentRedirect(workHistoryHref(work.id, work.name));
}
