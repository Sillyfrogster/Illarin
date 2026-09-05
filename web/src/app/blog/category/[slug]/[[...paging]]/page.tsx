import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { ScopedArchive } from "@/components/publication/Archive";
import { fetchPostArchive } from "@/lib/api/query";
import { archivePage, blogMetadata } from "@/lib/publication-metadata";

export async function generateMetadata({
  params,
}: PageProps<"/blog/category/[slug]/[[...paging]]">): Promise<Metadata> {
  const { slug, paging } = await params;
  const page = archivePage(paging);
  if (page === null) return { title: "Not found" };
  const archive = await fetchPostArchive({ page, category: slug });
  if (!archive?.category) return { title: "Not found" };
  return blogMetadata(
    archive.category.label,
    `Everything Illarin has published under ${archive.category.label}.`,
    page === 1
      ? `/blog/category/${slug}`
      : `/blog/category/${slug}/page/${page}`,
  );
}

export default async function CategoryArchivePage({
  params,
}: PageProps<"/blog/category/[slug]/[[...paging]]">) {
  const { slug, paging } = await params;
  const page = archivePage(paging);
  if (page === null) notFound();
  if (paging?.length === 2 && page === 1) {
    permanentRedirect(`/blog/category/${slug}`);
  }
  const archive = await fetchPostArchive({ page, category: slug });
  if (!archive?.category) notFound();
  if (page > 1 && archive.posts.length === 0) notFound();
  return (
    <ScopedArchive
      archive={archive}
      scope={{
        heading: archive.category.label,
        address: `/blog/category/${slug}`,
        narrowed: "category",
        home: null,
      }}
    />
  );
}
