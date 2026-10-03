import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";
import { SITE_DESCRIPTION } from "@/lib/site-metadata";
import { HomeSearch } from "./HomeSearch";

/** HomeHero is the line and search over the hall of windows, drawn as a CSS background so the Artwork switch stops it loading. */
export function HomeHero() {
  return (
    <section className="relative isolate -mt-(--header-height) pt-(--header-height)">
      <div
        aria-hidden="true"
        className="absolute inset-x-0 top-0 -z-10 h-[46rem] bg-[url(/home/atrium-day.webp)] bg-cover bg-[position:50%_22%] dark:bg-[url(/home/atrium-night.webp)] max-sm:h-[34rem]"
        data-artwork
      >
        <div className="absolute inset-0 bg-linear-to-b from-field/75 via-field/10 via-45% to-field" />
      </div>
      <div
        className={cn(
          shellClasses,
          "pt-14 pb-[21rem] text-center max-sm:pt-12 max-sm:pb-28",
        )}
      >
        <h1 className="mx-auto max-w-[34ch] font-display text-title font-medium tracking-tight text-balance text-ink">
          {SITE_DESCRIPTION}
        </h1>
        <div className="mx-auto mt-8 max-w-[36rem] text-left">
          <HomeSearch />
        </div>
      </div>
    </section>
  );
}
