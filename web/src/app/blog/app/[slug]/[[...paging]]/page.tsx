import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { ScopedArchive } from "@/components/publication/Archive";
import { fetchPostArchive } from "@/lib/api/query";
import { archivePage, blogMetadata } from "@/lib/publication-metadata";

export async function generateMetadata({
  params,
}: PageProps<"/blog/app/[slug]/[[...paging]]">): Promise<Metadata> {
  const { slug, paging } = await params;
  const page = archivePage(paging);
  if (page === null) return { title: "Not found" };
  const archive = await fetchPostArchive({ page, app: slug });
  if (!archive?.app) return { title: "Not found" };
  return blogMetadata(
    archive.app.name,
    `Everything Illarin has published about ${archive.app.name}.`,
    page === 1 ? `/blog/app/${slug}` : `/blog/app/${slug}/page/${page}`,
  );
}

export default async function AppArchivePage({
  params,
}: PageProps<"/blog/app/[slug]/[[...paging]]">) {
  const { slug, paging } = await params;
  const page = archivePage(paging);
  if (page === null) notFound();
  if (paging?.length === 2 && page === 1) {
    permanentRedirect(`/blog/app/${slug}`);
  }
  const archive = await fetchPostArchive({ page, app: slug });
  if (!archive?.app) notFound();
  if (page > 1 && archive.posts.length === 0) notFound();
  return (
    <ScopedArchive
      archive={archive}
      scope={{
        heading: archive.app.name,
        address: `/blog/app/${slug}`,
        narrowed: "app",
        home: { label: `Visit ${archive.app.name}`, href: archive.app.home },
      }}
    />
  );
}
