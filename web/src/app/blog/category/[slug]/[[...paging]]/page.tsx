import type { Metadata } from "next";
import { ScopedArchive } from "@/components/publication/Archive";
import { archiveDescription, blogMetadata } from "@/lib/publication-metadata";
import { scopedArchive } from "@/lib/scoped-archive";

export async function generateMetadata({
  params,
}: PageProps<"/blog/category/[slug]/[[...paging]]">): Promise<Metadata> {
  const { slug, paging } = await params;
  const { found, address, canonical } = await scopedArchive(
    "category",
    slug,
    paging,
  );
  return blogMetadata(
    found.label,
    archiveDescription("category", found.label),
    canonical,
    address,
  );
}

export default async function CategoryArchivePage({
  params,
}: PageProps<"/blog/category/[slug]/[[...paging]]">) {
  const { slug, paging } = await params;
  const { archive, found, address } = await scopedArchive(
    "category",
    slug,
    paging,
  );
  return (
    <ScopedArchive
      archive={archive}
      scope={{
        kind: "Category",
        heading: found.label,
        statement: archiveDescription("category", found.label),
        address,
        home: null,
      }}
    />
  );
}
