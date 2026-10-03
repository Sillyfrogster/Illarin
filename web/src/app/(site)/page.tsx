import { cookies } from "next/headers";
import { cache, Suspense } from "react";
import { AppReads } from "@/components/home/AppReads";
import { Closing, PublishBand } from "@/components/home/Closing";
import { CoverRail } from "@/components/home/CoverRail";
import { HomeHero } from "@/components/home/HomeHero";
import { HEADING } from "@/components/home/heading";
import { Statement } from "@/components/home/Statement";
import { TypeTiles } from "@/components/home/TypeTiles";
import { shellClasses } from "@/components/layout/Shell";
import { fetchWorks } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { WORK_TYPES } from "@/lib/work-types";

const PER_TYPE = 12;

/** HOME_TYPES leaves out packs, which only Lumiverse reads. */
const HOME_TYPES = WORK_TYPES.filter((type) => type !== "pack");

export default function HomePage() {
  return (
    <>
      <HomeHero />
      <Suspense fallback={<div className="h-[clamp(19rem,36vh,27rem)]" />}>
        <Rail />
      </Suspense>
      <Statement />
      <Suspense fallback={null}>
        <Types />
      </Suspense>
      <section
        aria-labelledby="home-apps"
        className={cn(shellClasses, "pt-chapter")}
      >
        <AppReads />
      </section>
      <PublishBand />
      <Closing />
    </>
  );
}

/** lists fetches each type's most downloaded works once per request, shared by the rail and the type rows. */
const lists = cache(async () => {
  const cookie = (await cookies()).toString();
  return Promise.all(
    HOME_TYPES.map(async (type) => ({
      type,
      page: await fetchWorks(
        { type, sort: "downloads", limit: PER_TYPE },
        cookie,
        AbortSignal.timeout(6000),
      ).catch(() => null),
    })),
  );
});

async function Rail() {
  const found = await lists();
  const works = found
    .flatMap(({ page }) =>
      (page?.items ?? []).filter(
        (work) =>
          work.cover && (!work.isNsfw || page?.nsfwPreference === "shown"),
      ),
    )
    .sort((a, b) => b.downloadCount - a.downloadCount);
  return <CoverRail works={works} />;
}

async function Types() {
  const found = await lists();
  const first = found.find((list) => list.page)?.page;
  if (!first) return null;
  const tiles = HOME_TYPES.map((type) => ({
    type,
    count: first.types.find((one) => one.value === type)?.count ?? 0,
    works: (found.find((list) => list.type === type)?.page?.items ?? []).filter(
      (work) => !work.isNsfw,
    ),
  })).filter((tile) => tile.count > 0);
  return (
    <section
      aria-labelledby="home-types"
      className={cn(shellClasses, "pt-chapter")}
    >
      <h2 className={cn(HEADING, "max-w-[26ch]")} id="home-types">
        Everything the hobby makes
      </h2>
      <div className="mt-section">
        <TypeTiles tiles={tiles} />
      </div>
    </section>
  );
}
