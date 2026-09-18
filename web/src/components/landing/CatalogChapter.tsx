import { ArrowUpRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import { LineLink } from "@/components/ui/line-link";
import type { BrowsePage } from "@/lib/api/query";
import { assetHref } from "@/lib/asset-url";
import { cn } from "@/lib/cn";
import { KIND_LABELS } from "@/lib/kinds";
import { CatalogRetry } from "./CatalogRetry";
import { Reveal } from "./LandingMotion";

function CatalogHeading() {
  return (
    <div className="mb-10 flex flex-wrap items-end justify-between gap-6">
      <div>
        <p className="mb-3 font-ui text-ui text-accent">Recently published</p>
        <h2
          id="landing-catalog-title"
          className="font-display text-[clamp(2.7rem,4.8vw,5rem)] leading-[1.05] font-medium tracking-[-.045em]"
        >
          New in the catalog.
        </h2>
      </div>
      <LineLink href="/browse" className="text-ink">
        See the full catalog{" "}
        <ArrowUpRight className="ml-2 inline size-4" aria-hidden="true" />
      </LineLink>
    </div>
  );
}

const GRID =
  "m-0 grid list-none grid-cols-2 items-start gap-x-5 gap-y-10 p-0 md:grid-cols-3 lg:grid-cols-5 lg:gap-6";

export function CatalogChapter({ page }: { page: BrowsePage | null }) {
  const items = page?.items ?? [];
  return (
    <section
      id="recent"
      className="relative scroll-mt-28 pb-20 sm:pb-32"
      aria-labelledby="landing-catalog-title"
    >
      <Shell>
        <CatalogHeading />
        <div className="min-h-[460px]">
          {items.length ? (
            <ul className={GRID}>
              {items.map((asset, index) => (
                <li key={asset.id} className="min-w-0">
                  <Reveal delay={index * 0.05}>
                    <Link
                      href={assetHref(asset.id, asset.name)}
                      className="group block min-w-0 rounded-control text-ink outline-offset-4"
                    >
                      <div className="relative flex aspect-[4/5] items-center justify-center overflow-hidden rounded-plate bg-deep">
                        {asset.cover ? (
                          <Image
                            src={asset.cover.url}
                            alt=""
                            width={asset.cover.width}
                            height={asset.cover.height}
                            sizes="(max-width: 767px) 45vw, (max-width: 1023px) 30vw, 18vw"
                            unoptimized
                            className={cn(
                              "h-full w-full object-contain",
                              asset.isNsfw &&
                                page?.nsfwPreference !== "shown" &&
                                "blur-xl",
                            )}
                          />
                        ) : (
                          <div className="flex h-full w-full flex-col justify-between p-4 sm:p-5">
                            <span className="font-ui text-meta text-mute">
                              {KIND_LABELS[asset.type]}
                            </span>
                            <span className="break-words font-display text-[clamp(1.4rem,2vw,2rem)] leading-tight text-ink">
                              {KIND_LABELS[asset.type]}
                              <span className="mt-4 block font-ui text-meta text-mute">
                                No cover image
                              </span>
                            </span>
                          </div>
                        )}
                        <span className="absolute top-3 right-3 flex size-8 items-center justify-center rounded-full bg-plane text-ink transition-colors group-hover:bg-action group-hover:text-on-accent">
                          <ArrowUpRight className="size-4" aria-hidden="true" />
                        </span>
                      </div>
                      <div className="pt-5">
                        <p className="font-ui text-meta text-accent">
                          {KIND_LABELS[asset.type]}
                        </p>
                        <h3 className="mt-2 break-words font-display text-[clamp(1.15rem,1.5vw,1.5rem)] leading-[1.2] font-medium tracking-tight group-hover:text-accent">
                          {asset.name || "Untitled"}
                        </h3>
                        <p className="mt-3 break-all text-meta text-mute">
                          @{asset.creator}
                        </p>
                        {asset.isNsfw ? (
                          <p className="mt-1 text-meta text-mute">
                            {page?.nsfwPreference === "shown"
                              ? "Adult"
                              : "Adult · blurred"}
                          </p>
                        ) : null}
                      </div>
                    </Link>
                  </Reveal>
                </li>
              ))}
            </ul>
          ) : (
            <div className="flex min-h-[460px] flex-col items-start justify-center rounded-plate bg-deep p-8 sm:p-12">
              <h3 className="font-display text-title font-medium">
                {!page
                  ? "Recent assets could not load."
                  : page.emptyState === "suppressed" || page.suppressed > 0
                    ? "Recent assets are hidden."
                    : "No assets published yet."}
              </h3>
              <p
                className="mt-4 max-w-[580px] text-prose text-mute"
                role={!page ? "status" : undefined}
              >
                {!page
                  ? "We couldn’t load recent creations. Try again, or head into Browse."
                  : page.emptyState === "suppressed" || page.suppressed > 0
                    ? "Published work is outside your content preference. You can change what you see in Browse."
                    : "Import a file or create a draft to publish the first asset."}
              </p>
              <div className="mt-6 flex flex-wrap gap-3">
                {!page ? <CatalogRetry /> : null}
                <Button asChild variant="outline">
                  <Link
                    href={
                      page?.emptyState === "catalog" ? "/upload" : "/browse"
                    }
                  >
                    {page?.emptyState === "catalog"
                      ? "Share your work"
                      : "Browse the catalog"}
                    <ArrowUpRight aria-hidden="true" />
                  </Link>
                </Button>
              </div>
            </div>
          )}
        </div>
        {page && page.suppressed > 0 && items.length > 0 ? (
          <p className="mt-6 text-meta text-mute">
            {page.suppressed}{" "}
            {page.suppressed === 1 ? "creation is" : "creations are"} outside
            your content preference.
          </p>
        ) : null}
      </Shell>
    </section>
  );
}

export function CatalogLoading() {
  return (
    <section
      className="pb-20 sm:pb-32"
      aria-labelledby="landing-catalog-title"
      aria-busy="true"
    >
      <Shell>
        <CatalogHeading />
        <output className="sr-only">Finding recent creations…</output>
        <div className={cn(GRID, "min-h-[460px]")} aria-hidden="true">
          {[0, 1, 2, 3, 4].map((index) => (
            <div key={index}>
              <div className="aspect-[4/5] rounded-plate bg-deep" />
              <div className="mt-5 h-4 w-1/3 rounded-control bg-deep" />
              <div className="mt-3 h-12 rounded-control bg-deep" />
              <div className="mt-3 h-4 w-2/3 rounded-control bg-deep" />
            </div>
          ))}
        </div>
      </Shell>
    </section>
  );
}
