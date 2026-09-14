import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { DocPage } from "@/components/docs/DocPage";
import { DOCS, findDocCollection, findDocPage } from "@/lib/docs/collections";
import { loadDoc } from "@/lib/docs/load-doc";
import { pageMetadata } from "@/lib/site-metadata";

export const dynamicParams = false;

type Params = { collection: string; page?: string[] };

export function generateStaticParams(): Params[] {
  return DOCS.flatMap((collection) =>
    collection.pages.map((page) => ({
      collection: collection.directory,
      page: page.slug ? [page.slug] : [],
    })),
  );
}

function pageOf(params: Params) {
  const collection = findDocCollection(params.collection);
  const [slug = "", ...rest] = params.page ?? [];
  if (!collection || rest.length > 0) return null;
  const page = findDocPage(collection, slug);
  return page ? { collection, page } : null;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<Params>;
}): Promise<Metadata> {
  const found = pageOf(await params);
  if (!found) return {};
  const { collection, page } = found;
  const title =
    page.slug === "" ? collection.name : `${page.title} · ${collection.name}`;
  return pageMetadata(title, page.summary);
}

export default async function DeveloperDocPage({
  params,
}: {
  params: Promise<Params>;
}) {
  const found = pageOf(await params);
  if (!found) notFound();
  const doc = await loadDoc(found.collection, found.page);
  return <DocPage collection={found.collection} doc={doc} page={found.page} />;
}
