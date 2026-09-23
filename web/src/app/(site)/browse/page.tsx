import type { Metadata } from "next";
import { cookies } from "next/headers";
import Image from "next/image";
import { BrowseSurface } from "@/components/browse/BrowseSurface";
import { fetchWorks } from "@/lib/api/query";
import { readBrowseFilters } from "@/lib/browse-url";
import { pageMetadata } from "@/lib/site-metadata";
import { TYPE_PLURALS } from "@/lib/work-types";

export async function generateMetadata({
  searchParams,
}: PageProps<"/browse">): Promise<Metadata> {
  const filters = readBrowseFilters(await searchParams);
  const subject = filters.type
    ? TYPE_PLURALS[filters.type].toLowerCase()
    : "characters, lorebooks, presets, themes and extensions";

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
    <div className="relative isolate">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-0 top-0 -z-1 h-[22rem] overflow-hidden [mask-image:linear-gradient(to_bottom,#000_20%,transparent)]"
      >
        <Image
          alt=""
          className="hidden object-cover object-[50%_62%] opacity-30 dark:block"
          fill
          priority
          sizes="100vw"
          src="/landing/flight/gallery.webp"
        />
        <Image
          alt=""
          className="object-cover object-[50%_40%] opacity-25 dark:hidden"
          fill
          priority
          sizes="100vw"
          src="/landing/flight/kingdom-distance.webp"
        />
      </div>
      <BrowseSurface
        filters={filters}
        heading="Browse"
        initialPage={initialPage}
        search={{
          hint: "Try tag:fantasy or author:handle",
          label: "Search works",
          placeholder: "Search works",
        }}
        showHeading
      />
    </div>
  );
}
