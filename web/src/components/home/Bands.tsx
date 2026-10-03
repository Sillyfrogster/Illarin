import { SiDiscord } from "@icons-pack/react-simple-icons";
import { Bell, LockKeyhole, Repeat } from "lucide-react";
import Link from "next/link";
import { TileArt } from "@/components/browse/WorkTile";
import { shellClasses } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { BLOG_HOME } from "@/lib/blog-paths";
import { cn } from "@/lib/cn";
import { DISCORD_INVITE } from "@/lib/contact";
import { workDisplayName } from "@/lib/work-name";

/** UpdatesPreview is a followed work's new-version notice next to the version line a connected app sees. */
export function UpdatesPreview({
  preference,
  work,
}: {
  preference: NsfwPreference;
  work: BrowseWork;
}) {
  const name = workDisplayName(work.name);
  return (
    <div className="grid grid-cols-1 gap-3 rounded-card bg-inset p-6 sm:p-8">
      <div className="flex items-center gap-4 rounded-control bg-plane p-3 ring-1 ring-ink/6">
        <div className="w-12 shrink-0">
          <TileArt eager={false} preference={preference} work={work} />
        </div>
        <div className="min-w-0 flex-1">
          <p className="flex items-center gap-1.5 text-meta text-mute">
            <Bell aria-hidden="true" className="size-3.5 text-accent" />
            New version
          </p>
          <p className="truncate text-ui font-medium text-ink">{name}</p>
          <p className="truncate text-meta text-mute">
            @{work.creator} published a new version, with notes on what changed.
          </p>
        </div>
      </div>
      <div className="flex items-center gap-3 rounded-control bg-plane p-3 text-meta text-mute ring-1 ring-ink/6">
        <Repeat aria-hidden="true" className="size-4 shrink-0 text-accent" />
        Installed on Lumiverse, and a newer version exists here.
      </div>
    </div>
  );
}

/** PublishBand is the watcher at her desk beside what publishing on Illarin does for a creator. */
export function PublishBand() {
  return (
    <section
      aria-labelledby="home-publish"
      className={cn(
        shellClasses,
        "grid grid-cols-1 items-center gap-group pt-chapter md:grid-cols-2 md:gap-section [&>*]:min-w-0",
      )}
    >
      <div
        aria-hidden="true"
        className="aspect-5/4 rounded-card bg-inset bg-[url(/home/watcher-studio.webp)] bg-cover bg-[position:50%_15%]"
        data-artwork
      />
      <div>
        <h2
          className="font-display text-title font-medium tracking-tight text-ink"
          id="home-publish"
        >
          Publish your work
        </h2>
        <p className="mt-2 text-lede text-mute">
          Upload a card, lorebook, preset, theme or extension file. Illarin
          builds its page and keeps every version.
        </p>
        <ul className="mt-6 grid list-none gap-3 p-0 text-ui text-ink">
          <li className="flex items-start gap-3">
            <LockKeyhole
              aria-hidden="true"
              className="mt-1 size-4 shrink-0 text-accent"
            />
            Drafts stay private until you publish them.
          </li>
          <li className="flex items-start gap-3">
            <Repeat
              aria-hidden="true"
              className="mt-1 size-4 shrink-0 text-accent"
            />
            Readers get the format their app reads, whatever you uploaded.
          </li>
        </ul>
        <Button asChild className="mt-8" variant="primary">
          <Link href="/upload">Publish</Link>
        </Button>
      </div>
    </section>
  );
}

/** CreatorList links to the creators behind this month's popular works, each shown with their top cover. */
export function CreatorList({
  creators,
}: {
  creators: { handle: string; work: BrowseWork }[];
}) {
  return (
    <div className="grid gap-6">
      <ul className="m-0 grid list-none grid-cols-2 gap-4 p-0 sm:grid-cols-3 lg:grid-cols-6">
        {creators.map(({ handle, work }) => (
          <li key={handle}>
            <Link
              className="group/creator flex items-center gap-3 rounded-card bg-inset p-3 text-ink transition-colors duration-80 hover:bg-plane"
              href={`/@${handle}`}
            >
              <span
                aria-hidden="true"
                className="grid size-11 shrink-0 place-items-center rounded-full bg-plane bg-cover bg-top text-ui font-medium text-mute uppercase ring-1 ring-ink/8"
                style={
                  work.cover && !work.isNsfw
                    ? { backgroundImage: `url(${work.cover.url})` }
                    : undefined
                }
              >
                {work.cover && !work.isNsfw ? null : handle[0]}
              </span>
              <span className="min-w-0 truncate text-ui font-medium">
                @{handle}
              </span>
            </Link>
          </li>
        ))}
      </ul>
      <div className="flex flex-wrap gap-3">
        <Button asChild>
          <a href={DISCORD_INVITE} rel="noopener" target="_blank">
            <SiDiscord aria-hidden="true" className="size-4" title="" />
            Discord
          </a>
        </Button>
        <Button asChild>
          <Link href={BLOG_HOME}>Blog</Link>
        </Button>
      </div>
    </div>
  );
}

/** CloseBand ends the page with the watcher reading and the same two ways in as the top. */
export function CloseBand() {
  return (
    <section
      aria-labelledby="home-close"
      className={cn(shellClasses, "pt-chapter")}
    >
      <div className="relative isolate flex min-h-80 items-center overflow-hidden rounded-card bg-inset p-8 text-ink sm:p-14 [html:not([data-artwork=off])_&]:bg-[#161616] [html:not([data-artwork=off])_&]:text-over">
        <div
          aria-hidden="true"
          className="absolute inset-y-0 right-0 -z-10 w-1/2 bg-[url(/reading/watcher-reading.webp)] bg-contain bg-right bg-no-repeat max-md:w-full max-md:opacity-40"
          data-artwork
        />
        <div>
          <h2
            className="font-display text-title font-medium tracking-tight"
            id="home-close"
          >
            Browse every work
          </h2>
          <div className="mt-6 flex flex-wrap gap-3">
            <Button asChild variant="primary">
              <Link href="/browse">Browse</Link>
            </Button>
            <Button asChild>
              <Link href="/upload">Publish</Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
