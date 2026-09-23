import { notFound, permanentRedirect } from "next/navigation";
import { cache } from "react";
import { fetchPostArchive, type PostArchive } from "@/lib/api/query";
import { archivePage, archivePath, pageAddress } from "@/lib/blog-paths";

const loadArchive = cache(async (slug: string, page: number) =>
  fetchPostArchive({ page, category: slug }),
);

export async function categoryArchive(
  slug: string,
  paging: string[] | undefined,
): Promise<{
  archive: PostArchive;
  found: NonNullable<PostArchive["category"]>;
  address: string;
  canonical: string;
}> {
  const page = archivePage(paging);
  if (page === null) notFound();
  const address = archivePath("category", slug);
  if (paging?.length === 2 && page === 1) permanentRedirect(address);
  const archive = await loadArchive(slug, page);
  const found = archive?.category;
  if (!archive || !found) notFound();
  if (page > 1 && archive.posts.length === 0) notFound();
  return { archive, found, address, canonical: pageAddress(address, page) };
}
