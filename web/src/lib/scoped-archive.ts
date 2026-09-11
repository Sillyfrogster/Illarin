import { notFound, permanentRedirect } from "next/navigation";
import { cache } from "react";
import { fetchPostArchive, type PostArchive } from "@/lib/api/query";
import { archivePage, archivePath, pageAddress } from "@/lib/blog-paths";

type Narrowing = "category" | "app";

const loadArchive = cache(
  async (scope: Narrowing, slug: string, page: number) =>
    fetchPostArchive({ page, [scope]: slug }),
);

export async function scopedArchive<Scope extends Narrowing>(
  scope: Scope,
  slug: string,
  paging: string[] | undefined,
): Promise<{
  archive: PostArchive;
  found: NonNullable<PostArchive[Scope]>;
  address: string;
  canonical: string;
}> {
  const page = archivePage(paging);
  if (page === null) notFound();
  const address = archivePath(scope, slug);
  if (paging?.length === 2 && page === 1) permanentRedirect(address);
  const archive = await loadArchive(scope, slug, page);
  const found = archive?.[scope];
  if (!archive || !found) notFound();
  if (page > 1 && archive.posts.length === 0) notFound();
  return { archive, found, address, canonical: pageAddress(address, page) };
}
