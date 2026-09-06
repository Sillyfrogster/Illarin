import type { Metadata } from "next";
import { ScopedArchive } from "@/components/publication/Archive";
import { archiveDescription, blogMetadata } from "@/lib/publication-metadata";
import { scopedArchive } from "@/lib/scoped-archive";

export async function generateMetadata({
  params,
}: PageProps<"/blog/app/[slug]/[[...paging]]">): Promise<Metadata> {
  const { slug, paging } = await params;
  const { found, address, canonical } = await scopedArchive(
    "app",
    slug,
    paging,
  );
  return blogMetadata(
    found.name,
    archiveDescription("app", found.name),
    canonical,
    address,
  );
}

export default async function AppArchivePage({
  params,
}: PageProps<"/blog/app/[slug]/[[...paging]]">) {
  const { slug, paging } = await params;
  const { archive, found, address } = await scopedArchive("app", slug, paging);
  return (
    <ScopedArchive
      archive={archive}
      scope={{
        kind: "Publication app",
        heading: found.name,
        address,
        home: found.home,
      }}
    />
  );
}
