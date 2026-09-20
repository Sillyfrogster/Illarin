import type { Metadata } from "next";
import { ScopedArchive } from "@/components/blog/Archive";
import { archiveDescription, blogMetadata } from "@/lib/blog-metadata";
import { categoryArchive } from "@/lib/category-archive";

export async function generateMetadata({
  params,
}: PageProps<"/blog/category/[slug]/[[...paging]]">): Promise<Metadata> {
  const { slug, paging } = await params;
  const { found, address, canonical } = await categoryArchive(slug, paging);
  return blogMetadata(
    found.label,
    archiveDescription(found.label),
    canonical,
    address,
  );
}

export default async function CategoryArchivePage({
  params,
}: PageProps<"/blog/category/[slug]/[[...paging]]">) {
  const { slug, paging } = await params;
  const { archive, found, address } = await categoryArchive(slug, paging);
  return (
    <ScopedArchive
      archive={archive}
      scope={{
        kind: "Category",
        heading: found.label,
        statement: archiveDescription(found.label),
        address,
        home: null,
      }}
    />
  );
}
