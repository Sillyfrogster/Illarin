import type { Metadata } from "next";
import { cookies } from "next/headers";
import { BrowseSurface } from "@/components/browse/BrowseSurface";
import { fetchWorks } from "@/lib/api/query";
import { readBrowseFilters } from "@/lib/browse-url";
import { pageMetadata } from "@/lib/site-metadata";
import { TYPE_LABELS } from "@/lib/work-types";
import { BrowseThreshold } from "./BrowseThreshold";

export async function generateMetadata({
  searchParams,
}: PageProps<"/browse">): Promise<Metadata> {
  const filters = readBrowseFilters(await searchParams);
  const subject = filters.type
    ? `${TYPE_LABELS[filters.type].toLowerCase()}s`
    : "characters, lorebooks, presets, themes and packs";

  if (filters.q) {
    return pageMetadata(
      `${filters.q} · Browse`,
      `Illarin ${subject} matching ${filters.q}.`,
    );
  }
  return pageMetadata("Browse", `Every one of Illarin's ${subject}.`);
}

export default async function BrowsePage({
  searchParams,
}: PageProps<"/browse">) {
  const filters = readBrowseFilters(await searchParams);
  const cookie = (await cookies()).toString();
  const initialPage = await fetchWorks({ ...filters, limit: 24 }, cookie).catch(
    () => null,
  );

  return (
    <>
      <BrowseThreshold filters={filters} />
      <BrowseSurface
        filters={filters}
        heading="Works"
        initialPage={initialPage}
      />
    </>
  );
}
