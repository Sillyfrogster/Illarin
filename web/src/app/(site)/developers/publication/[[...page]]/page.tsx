import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { DocPage } from "@/components/docs/DocPage";
import { findDocPage, PUBLICATION_DOCS } from "@/lib/docs/collections";
import { loadDoc } from "@/lib/docs/load-doc";
import { pageMetadata } from "@/lib/site-metadata";

export const dynamicParams = false;

type Params = { page?: string[] };

export function generateStaticParams(): Params[] {
  return PUBLICATION_DOCS.pages.map((page) => ({
    page: page.slug ? [page.slug] : [],
  }));
}

function pageOf(params: Params) {
  const [slug = "", ...rest] = params.page ?? [];
  if (rest.length > 0) return null;
  return findDocPage(PUBLICATION_DOCS, slug);
}

export async function generateMetadata({
  params,
}: {
  params: Promise<Params>;
}): Promise<Metadata> {
  const page = pageOf(await params);
  if (!page) return {};
  const title =
    page.slug === ""
      ? PUBLICATION_DOCS.name
      : `${page.title} · ${PUBLICATION_DOCS.name}`;
  return pageMetadata(title, page.summary);
}

export default async function PublicationDocPage({
  params,
}: {
  params: Promise<Params>;
}) {
  const page = pageOf(await params);
  if (!page) notFound();
  const doc = await loadDoc(PUBLICATION_DOCS, page);
  return <DocPage collection={PUBLICATION_DOCS} doc={doc} page={page} />;
}
