import { cookies } from "next/headers";
import Link from "next/link";
import { Suspense } from "react";
import { BrowseRetry } from "@/components/browse/BrowseRetry";
import { Message } from "@/components/browse/BrowseStates";
import { AppDemo } from "@/components/home/AppDemo";
import {
  CloseBand,
  CreatorList,
  PublishBand,
  UpdatesPreview,
} from "@/components/home/Bands";
import { HomeHero } from "@/components/home/HomeHero";
import { PopularShelf } from "@/components/home/PopularShelf";
import { HOME_TYPES, type Home, homeRows } from "@/components/home/rows";
import { Section, Split } from "@/components/home/Section";
import { TypeTiles } from "@/components/home/TypeTiles";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { type BrowseWork, fetchWorks } from "@/lib/api/query";

const ROW_LENGTH = 10;
const CREATORS = 6;

export default function HomePage() {
  return (
    <>
      <HomeHero />
      <Suspense fallback={<ShelfLoading />}>
        <Bands />
      </Suspense>
    </>
  );
}

async function Bands() {
  const cookie = (await cookies()).toString();
  const lists = await Promise.all(
    HOME_TYPES.map(async (type) => ({
      type,
      page: await fetchWorks(
        { type, sort: "downloads", limit: ROW_LENGTH },
        cookie,
        AbortSignal.timeout(6000),
      ).catch(() => null),
    })),
  );
  const home = homeRows(lists);
  const first = lists.find((list) => list.page)?.page;
  const preference = first?.nsfwPreference ?? "blurred";
  const works = home.rows.flatMap((row) => row.page.items);
  const shown = works.filter((work) => work.cover && !work.isNsfw);
  const tiles = HOME_TYPES.map((type) => ({
    type,
    count: first?.types.find((one) => one.value === type)?.count ?? 0,
    covers: (
      lists.find((list) => list.type === type)?.page?.items ?? []
    ).filter((work) => work.cover && !work.isNsfw),
  })).filter((tile) => tile.count > 0);
  const top = shown[0] ?? works[0];

  return (
    <>
      <Section
        className="pt-section"
        id="home-popular"
        title="Popular this month"
      >
        {home.state === "rows" ? (
          <PopularShelf rows={home.rows} />
        ) : (
          <ShelfMessage state={home.state} />
        )}
        {home.someHidden ? (
          <p className="text-meta text-mute">Some adult works are hidden.</p>
        ) : null}
      </Section>
      {tiles.length ? (
        <Section id="home-types" title="Browse by type">
          <TypeTiles tiles={tiles} />
        </Section>
      ) : null}
      {top ? (
        <Split
          id="home-apps"
          title="Made for your app"
          visual={<AppDemo preference={preference} work={top} />}
        >
          <p>
            Press Download for SillyTavern, RisuAI or Lumiverse and Illarin
            picks the file that app reads.
          </p>
          <p>
            Connect an app and Send puts the work straight into it.{" "}
            <Link className="font-medium" href="/settings#connected-apps">
              Connect an app
            </Link>
          </p>
        </Split>
      ) : null}
      {top ? (
        <Split
          id="home-updates"
          reverse
          title="Stays up to date"
          visual={<UpdatesPreview preference={preference} work={top} />}
        >
          <p>
            Follow a work to hear when its creator publishes a new version, with
            notes on what changed.
          </p>
          <p>
            With a connected app, the work page shows the version you have and
            says when a newer one is out.
          </p>
        </Split>
      ) : null}
      <PublishBand />
      {works.length ? (
        <Section
          id="home-creators"
          line="Find them and other readers on Discord."
          title="Creators on Illarin"
        >
          <CreatorList creators={creatorsOf(works)} />
        </Section>
      ) : null}
      <CloseBand />
    </>
  );
}

/** creatorsOf lists the first few distinct creators in popularity order, each with their top work. */
function creatorsOf(works: BrowseWork[]) {
  const seen = new Map<string, BrowseWork>();
  for (const work of works)
    if (!seen.has(work.creator)) seen.set(work.creator, work);
  return [...seen]
    .slice(0, CREATORS)
    .map(([handle, work]) => ({ handle, work }));
}

function ShelfMessage({ state }: { state: Exclude<Home["state"], "rows"> }) {
  if (state === "failed")
    return (
      <Message
        action={<BrowseRetry />}
        body="Try again, or go to Browse."
        title="Popular works could not load."
      />
    );
  if (state === "hidden")
    return (
      <Message
        action={<Go href="/browse" label="Browse" />}
        body="Everything here is adult work, and you hide adult work. Change that in Browse."
        title="Adult works are hidden"
      />
    );
  if (state === "empty")
    return (
      <Message
        action={<Go href="/upload" label="Publish" />}
        body="Upload a character or a lorebook and it shows up here."
        title="Nothing published yet"
      />
    );
  return <Go href="/browse" label="Browse" />;
}

function Go({ href, label }: { href: string; label: string }) {
  return (
    <Button asChild variant="primary">
      <Link href={href}>{label}</Link>
    </Button>
  );
}

function ShelfLoading() {
  return (
    <output
      aria-label="Loading works"
      className="mx-auto flex max-w-[var(--shell)] gap-4 overflow-hidden px-[var(--gutter)] pt-section"
    >
      {[0, 1, 2, 3, 4].map((slot) => (
        <Skeleton
          className="aspect-3/4 w-[44vw] shrink-0 rounded-card sm:w-[calc((100%-2*1rem)/3)] lg:w-[calc((100%-4*1rem)/5)]"
          key={slot}
        />
      ))}
    </output>
  );
}
