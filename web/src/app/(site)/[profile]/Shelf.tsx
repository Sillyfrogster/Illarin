"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { Pin } from "lucide-react";
import Link from "next/link";
import { type ReactNode, useId, useMemo } from "react";
import { BrowsePoster } from "@/components/browse/BrowsePoster";
import { BrowseSearch } from "@/components/browse/BrowseSearch";
import { BrowseLoading, GRID, Message } from "@/components/browse/BrowseStates";
import { useBrowseNavigation } from "@/components/browse/use-browse-navigation";
import { Button } from "@/components/ui/button";
import { Scroller } from "@/components/ui/scroller";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import {
  type BrowseCursor,
  type BrowseFilters,
  type BrowsePage,
  type BrowseType,
  type BrowseWork,
  fetchWorks,
  type NsfwPreference,
  workKeys,
} from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";
import { cn } from "@/lib/cn";
import { FEATURED_LIMIT } from "@/lib/profile-portfolio";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_PLURALS, WORK_TYPES } from "@/lib/work-types";

const PAGE = 24;

const TAB =
  "relative flex min-h-11 items-center gap-1.5 rounded-control px-3.5 font-ui text-ui font-medium whitespace-nowrap outline-offset-3 transition-colors duration-300 motion-reduce:transition-none";

export type Pinning = {
  featured: string[];
  toggle: (work: BrowseWork) => void;
  pending: boolean;
  trouble: string;
};

/** The creator's work, with featured first and the rest behind type tabs and a search */
export function Shelf({
  featured,
  filters,
  handle,
  initialPage,
  isOwner,
  name,
  firstSteps,
  pinning,
  published,
}: {
  featured: BrowseWork[];
  filters: BrowseFilters;
  handle: string;
  initialPage: BrowsePage | null;
  isOwner: boolean;
  name: string;
  firstSteps: ReactNode;
  pinning: Pinning | null;
  published: number;
}) {
  const basePath = `/@${handle}`;
  const { navigate, pending } = useBrowseNavigation(basePath);
  const panel = useId();

  const query = useInfiniteQuery({
    queryKey: workKeys.list(filters, undefined, handle),
    queryFn: ({ pageParam }) =>
      fetchWorks({
        ...filters,
        creator: handle,
        limit: PAGE,
        before: pageParam?.before,
        beforeId: pageParam?.beforeId,
      }),
    initialPageParam: null as BrowseCursor | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    initialData: initialPage
      ? { pages: [initialPage], pageParams: [null] }
      : undefined,
  });
  const pages = query.data?.pages;
  const overview = pages?.[0];
  const works = useMemo(
    () => pages?.flatMap((page) => page.items) ?? [],
    [pages],
  );
  const preference = overview?.nsfwPreference ?? "blurred";
  const narrowed = Boolean(filters.type || filters.q);
  const nothingAtAll =
    !narrowed && overview?.emptyState === "nothing_published";

  const tabs = [
    { key: "all", label: "All", count: overview?.allTypes },
    ...WORK_TYPES.flatMap((type) => {
      const held = overview?.types.find((one) => one.value === type);
      return held && held.count > 0
        ? [{ key: type, type, label: TYPE_PLURALS[type], count: held.count }]
        : [];
    }),
  ] as { key: string; type?: BrowseType; label: string; count?: number }[];

  const pinFor = (work: BrowseWork) =>
    pinning && !work.ownerState ? (
      <PinToggle pinning={pinning} work={work} />
    ) : undefined;

  return (
    <div className="min-w-0">
      <Featured
        featured={featured}
        isOwner={isOwner}
        pinFor={pinFor}
        preference={preference}
        published={published}
        trouble={pinning?.trouble ?? ""}
      />

      <section aria-labelledby={`${panel}-heading`} className="min-w-0">
        <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-4">
          <h2
            className="flex items-baseline gap-3 font-display text-[clamp(1.75rem,2.6vw,2.25rem)] leading-[1.1] font-medium tracking-[-0.035em] text-ink"
            id={`${panel}-heading`}
          >
            {isOwner ? "Your work" : "Work"}
            {overview?.allTypes ? (
              <span className="font-ui text-lede font-medium tracking-normal text-mute tabular-nums">
                {overview.allTypes}
              </span>
            ) : null}
          </h2>
          {nothingAtAll ? null : (
            <div className="w-full min-w-0 sm:w-[min(20rem,100%)]">
              <BrowseSearch
                id={`${panel}-search`}
                label={isOwner ? "Search your work" : `Search ${name}'s work`}
                onSearch={(q) => navigate({ ...filters, q })}
                placeholder={isOwner ? "Search your work" : "Search their work"}
                value={filters.q ?? ""}
              />
            </div>
          )}
        </div>

        {nothingAtAll || tabs.length < 3 ? null : (
          <nav aria-label="Type" className="mt-4 -ml-1 min-w-0">
            <Scroller buttonClassName="bottom-0.5">
              <TravellingHighlight
                chosen={filters.type ?? "all"}
                className="w-max"
                plateClassName="rounded-control"
              >
                <ul className="m-0 flex w-max list-none gap-1 p-1">
                  {tabs.map((tab) => {
                    const here = (filters.type ?? "all") === tab.key;
                    return (
                      <li key={tab.key}>
                        <Link
                          aria-current={here ? "page" : undefined}
                          className={cn(
                            TAB,
                            here
                              ? "text-on-accent"
                              : "text-mute hover:text-ink",
                          )}
                          data-cell={tab.key}
                          href={buildBrowseHref(
                            { ...filters, type: tab.type, facet: undefined },
                            basePath,
                          )}
                          onClick={(event) => {
                            event.preventDefault();
                            navigate({
                              ...filters,
                              type: tab.type,
                              facet: undefined,
                            });
                          }}
                        >
                          {tab.label}
                          {tab.count === undefined ? null : (
                            <span
                              className={cn(
                                "font-ui text-meta tabular-nums",
                                here ? "text-on-accent/80" : "text-mute",
                              )}
                            >
                              <span className="sr-only">, </span>
                              {tab.count}
                              <span className="sr-only">
                                {tab.count === 1 ? " work" : " works"}
                              </span>
                            </span>
                          )}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </TravellingHighlight>
            </Scroller>
          </nav>
        )}

        <div
          aria-busy={pending || query.isFetching || undefined}
          className={cn(
            "mt-8 transition-opacity duration-200 motion-reduce:transition-none",
            (pending || (query.isFetching && !query.isFetchingNextPage)) &&
              "opacity-60",
          )}
        >
          {query.isPending ? <BrowseLoading /> : null}

          {query.isError ? (
            <Message
              action={
                <Button onClick={() => void query.refetch()} variant="primary">
                  Try again
                </Button>
              }
              body="Check your connection, then try again."
              title="This work could not load"
            />
          ) : null}

          {!query.isPending && !query.isError && overview?.total === 0 ? (
            <Nothing
              clear={() => navigate({})}
              firstSteps={firstSteps}
              handle={handle}
              isOwner={isOwner}
              narrowed={narrowed}
              state={overview.emptyState}
              suppressed={overview.suppressed}
            />
          ) : null}

          {works.length ? (
            <ul className={GRID}>
              {works.map((work, index) => (
                <BrowsePoster
                  action={pinFor(work)}
                  byline={false}
                  eager={index < 5}
                  key={work.id}
                  preference={preference}
                  work={work}
                />
              ))}
            </ul>
          ) : null}

          {query.hasNextPage ? (
            <div className="mt-14 flex justify-center">
              <Button
                loading={query.isFetchingNextPage}
                onClick={() => void query.fetchNextPage()}
                size="large"
                variant="outline"
              >
                {query.isFetchingNextPage ? "Loading" : "Show more"}
              </Button>
            </div>
          ) : null}
        </div>
      </section>
    </div>
  );
}

/** Up to four works the creator chose to show first */
function Featured({
  featured,
  isOwner,
  pinFor,
  preference,
  published,
  trouble,
}: {
  featured: BrowseWork[];
  isOwner: boolean;
  pinFor: (work: BrowseWork) => ReactNode;
  preference: NsfwPreference;
  published: number;
  trouble: string;
}) {
  const heading = useId();
  if (featured.length === 0 && !(isOwner && published > 0)) return null;
  const open = isOwner ? FEATURED_LIMIT - featured.length : 0;

  return (
    <section aria-labelledby={heading} className="mb-14 min-w-0">
      <h2
        className="flex items-center gap-2.5 font-display text-section font-medium tracking-[-0.02em] text-ink"
        id={heading}
      >
        <Pin
          aria-hidden="true"
          className="size-4.5 text-accent"
          strokeWidth={1.9}
        />
        Featured
        {isOwner ? (
          <span className="font-ui text-meta font-medium tracking-normal text-mute tabular-nums">
            {featured.length} of {FEATURED_LIMIT}
          </span>
        ) : null}
      </h2>
      {trouble ? (
        <p className="mt-3 font-ui text-meta text-stop" role="alert">
          {trouble}
        </p>
      ) : null}

      {featured.length === 0 ? (
        <p className="mt-4 flex items-center gap-3 rounded-plate border border-dashed border-rule px-5 py-4 font-ui text-ui text-mute">
          <Pin
            aria-hidden="true"
            className="size-4 shrink-0 text-accent"
            strokeWidth={1.9}
          />
          Pin up to {FEATURED_LIMIT} published works below to show them here
          first.
        </p>
      ) : (
        <ul className="m-0 mt-5 grid list-none grid-cols-2 items-start gap-x-4 gap-y-9 p-0 sm:grid-cols-4 sm:gap-x-5">
          <AnimatePresence initial={false}>
            {featured.map((work, index) => (
              <BrowsePoster
                action={pinFor(work)}
                animated
                byline={false}
                eager={index < 4}
                key={work.id}
                preference={preference}
                work={work}
              />
            ))}
            {Array.from({ length: open }, (_, slot) => (
              <motion.li
                animate={{ opacity: 1 }}
                aria-hidden="true"
                className="grid aspect-5/6 place-items-center rounded-plate border border-dashed border-rule text-mute"
                exit={{ opacity: 0 }}
                initial={{ opacity: 0 }}
                key={`open-${featured.length + slot}`}
                layout
              >
                <Pin className="size-5 opacity-50" strokeWidth={1.6} />
              </motion.li>
            ))}
          </AnimatePresence>
        </ul>
      )}
    </section>
  );
}

/** The pin on an owner's card that puts the work in Featured, or takes it out. */
function PinToggle({ pinning, work }: { pinning: Pinning; work: BrowseWork }) {
  const still = useReducedMotion();
  const pinned = pinning.featured.includes(work.id);
  const full = !pinned && pinning.featured.length >= FEATURED_LIMIT;
  const name = workDisplayName(work.name);
  return (
    <button
      aria-label={pinned ? `Unfeature ${name}` : `Feature ${name}`}
      aria-pressed={pinned}
      className={cn(
        "grid size-11 place-items-center rounded-control backdrop-blur-sm transition duration-200 outline-offset-2 motion-reduce:transition-none",
        pinned
          ? "bg-action text-on-accent shadow-[0_6px_16px_-6px_var(--v-action)]"
          : "bg-plane/80 text-ink hover:bg-plane focus-visible:opacity-100 [@media(hover:hover)]:opacity-0",
        !pinned &&
          (full
            ? "opacity-40 group-hover:opacity-40"
            : "group-hover:opacity-100"),
      )}
      disabled={pinning.pending || full}
      onClick={() => pinning.toggle(work)}
      title={
        full
          ? `Up to ${FEATURED_LIMIT} featured`
          : pinned
            ? "Featured"
            : "Feature"
      }
      type="button"
    >
      <motion.span
        animate={{ scale: 1, rotate: 0 }}
        className="flex"
        initial={still ? false : { scale: 0.5, rotate: pinned ? -30 : 30 }}
        key={pinned ? "pinned" : "loose"}
        transition={{ type: "spring", stiffness: 500, damping: 22 }}
      >
        <Pin
          aria-hidden="true"
          className="size-4"
          fill={pinned ? "currentColor" : "none"}
          strokeWidth={1.9}
        />
      </motion.span>
    </button>
  );
}

function Nothing({
  clear,
  firstSteps,
  handle,
  isOwner,
  narrowed,
  state,
  suppressed,
}: {
  clear: () => void;
  firstSteps: ReactNode;
  handle: string;
  isOwner: boolean;
  narrowed: boolean;
  state: BrowsePage["emptyState"];
  suppressed: number;
}) {
  if (state === "suppressed") {
    return (
      <Message
        action={
          <Button asChild variant="primary">
            <Link href="/settings#browse">Change adult content</Link>
          </Button>
        }
        body={
          suppressed === 1
            ? "1 work is hidden by your adult content setting."
            : `${suppressed} works are hidden by your adult content setting.`
        }
        title="Hidden by your setting"
      />
    );
  }
  if (narrowed) {
    return (
      <Message
        action={
          <Button onClick={clear} variant="primary">
            Show all
          </Button>
        }
        body="Try a different search or another type."
        title="No matches"
      />
    );
  }
  if (isOwner) return firstSteps;
  return (
    <Message
      body={`Follow @${handle} to hear when they publish.`}
      title="Nothing published yet"
    />
  );
}
