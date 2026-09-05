import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { ScopedArchive } from "@/components/publication/Archive";
import { fetchPostArchive } from "@/lib/api/query";
import {
  BLOG_DESCRIPTION,
  BLOG_TITLE,
  blogMetadata,
  pageAddress,
} from "@/lib/publication-metadata";

export async function generateMetadata({
  params,
}: PageProps<"/blog/page/[number]">): Promise<Metadata> {
  const page = readPage((await params).number);
  return blogMetadata(
    `Page ${page}`,
    BLOG_DESCRIPTION,
    pageAddress("/blog", page),
  );
}

export default async function BlogArchivePage({
  params,
}: PageProps<"/blog/page/[number]">) {
  const page = readPage((await params).number);
  if (page === 1) permanentRedirect("/blog");
  const archive = await fetchPostArchive({ page });
  if (!archive || archive.posts.length === 0) notFound();
  return (
    <ScopedArchive
      archive={archive}
      scope={{
        kind: "Publication",
        heading: BLOG_TITLE,
        address: "/blog",
        home: null,
      }}
    />
  );
}

function readPage(number: string): number {
  if (!/^[1-9][0-9]*$/.test(number)) notFound();
  return Number(number);
}
