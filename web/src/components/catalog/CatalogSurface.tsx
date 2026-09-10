"use client";

import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { SlidersHorizontal, X } from "lucide-react";
import { useEffect, useId, useMemo, useState } from "react";
import { Shell } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import {
  assetKeys,
  type BrowseCursor,
  type BrowseFilters,
  type BrowsePage,
  fetchAssets,
  type NsfwVisibility,
  saveNsfwVisibility,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { narrowingsInForce } from "@/lib/catalog-narrowing";
import { cn } from "@/lib/cn";
import {
  readSessionVisibility,
  writeSessionVisibility,
} from "@/lib/nsfw-visibility";
import { CatalogPoster } from "./CatalogPoster";
import { CatalogSearch } from "./CatalogSearch";
import { KindRail } from "./KindRail";
import { RefinePanel } from "./RefinePanel";
import { useCatalogNavigation } from "./use-catalog-navigation";

const PAGE = 24;

const GRID =
  "m-0 grid list-none grid-cols-2 items-stretch gap-x-5 gap-y-10 p-0 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5";

/**
 * The catalog itself: what is in it, what it has been narrowed to, and the way
 * to narrow it further. Browse and a creator's page are the same surface over
 * different scopes.
 */
export function CatalogSurface({
  basePath = "/browse",
  creator,
  filters,
  heading,
  initialPage,
  search,
}: {
  basePath?: string;
  creator?: string;
  filters: BrowseFilters;
  heading: string;
  initialPage: BrowsePage | null;
  /** A creator's page carries its own search; Browse asks in its header instead. */
  search?: { label: string; placeholder: string };
}) {
  const queryClient = useQueryClient();
  const { account } = useAuth();
  const { navigate, pending } = useCatalogNavigation(basePath);
  const panel = useId();

  const [refining, setRefining] = useState(false);
  const [visibilityOverride, setVisibilityOverride] =
    useState<NsfwVisibility>();
  const [preferenceError, setPreferenceError] = useState("");
  const [savingPreference, setSavingPreference] = useState(false);
  const [dismissedSuppression, setDismissedSuppression] = useState<string>();

  useEffect(() => {
    if (account !== null) {
      setVisibilityOverride(undefined);
      return;
    }
    setVisibilityOverride(readSessionVisibility());
  }, [account]);

  const query = useInfiniteQuery({
    queryKey: assetKeys.list(filters, visibilityOverride, creator),
    queryFn: ({ pageParam }) =>
      fetchAssets({
        ...filters,
        creator,
        limit: PAGE,
        nsfw: visibilityOverride,
        before: pageParam?.before,
        beforeId: pageParam?.beforeId,
      }),
    initialPageParam: null as BrowseCursor | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    initialData:
      initialPage && visibilityOverride === undefined
        ? { pages: [initialPage], pageParams: [null] }
        : undefined,
  });

  const pages = query.data?.pages;
  const overview = pages?.[0];
  const assets = useMemo(
    () => pages?.flatMap((page) => page.items) ?? [],
    [pages],
  );
  const visibility = visibilityOverride ?? overview?.visibility ?? "blurred";
  const narrowings = narrowingsInForce(filters, overview ?? null);
  const suppressionKey =
    overview?.visibility === "hidden" && overview.suppressed > 0
      ? `${JSON.stringify(filters)}:${overview.suppressed}`
      : undefined;

  function openContentSetting() {
    setRefining(true);
    requestAnimationFrame(() => {
      const group = document.getElementById(`${panel}-adult`);
      group?.scrollIntoView({ block: "center" });
      group?.focus({ preventScroll: true });
    });
  }

  async function setPreference(next: NsfwVisibility) {
    if (account === undefined || savingPreference) return;
    setPreferenceError("");
    setSavingPreference(true);
    try {
      if (account) {
        await saveNsfwVisibility(next);
        await queryClient.invalidateQueries({ queryKey: assetKeys.all });
      } else {
        writeSessionVisibility(next);
        setVisibilityOverride(next);
      }
    } catch {
      setPreferenceError("That preference could not be saved. Try again.");
    } finally {
      setSavingPreference(false);
    }
  }

  return (
    <Shell as="section" aria-labelledby={`${panel}-heading`}>
      <h2 className="sr-only" id={`${panel}-heading`}>
        {heading}
      </h2>

      <KindRail basePath={basePath} filters={filters} navigate={navigate} />

      <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-4">
        <p aria-live="polite" className="order-1 font-ui text-ui text-mute">
          {overview ? (
            <>
              <span className="font-medium text-ink tabular-nums">
                {overview.total}
              </span>{" "}
              {overview.total === 1 ? "creation" : "creations"}
              <span aria-hidden="true"> · </span>Newest first
            </>
          ) : (
            "Counting the catalog…"
          )}
        </p>
        {search ? (
          <div className="order-3 w-full md:order-2 md:w-auto md:min-w-[17rem] md:max-w-[26rem] md:flex-1">
            <CatalogSearch
              id={`${panel}-search`}
              label={search.label}
              onSearch={(q) => navigate({ ...filters, q })}
              placeholder={search.placeholder}
              value={filters.q ?? ""}
            />
          </div>
        ) : null}
        <Button
          aria-controls={panel}
          aria-expanded={refining}
          className={cn("order-2 ml-auto md:order-3", refining && "bg-rule/45")}
          onClick={() => setRefining((open) => !open)}
          variant="secondary"
        >
          <SlidersHorizontal aria-hidden="true" />
          Refine
        </Button>
      </div>

      {narrowings.length ? (
        <ul className="mt-4 flex list-none flex-wrap items-center gap-2 p-0">
          {narrowings.map((one) => (
            <li key={one.id}>
              <button
                className="group inline-flex min-h-9 max-w-full items-center gap-2 rounded-control bg-accent-wash py-1 pr-2 pl-3 font-ui text-meta text-accent outline-offset-2 transition-colors duration-200 hover:bg-accent hover:text-on-accent motion-reduce:transition-none"
                onClick={() => navigate(one.without)}
                type="button"
              >
                <span className="min-w-0 truncate">
                  <span className="opacity-70">{one.group}: </span>
                  <span className="font-medium">{one.label}</span>
                </span>
                <X aria-hidden="true" className="size-3.5 shrink-0" />
                <span className="sr-only">Remove this</span>
              </button>
            </li>
          ))}
          {narrowings.length > 1 ? (
            <li>
              <button
                className="inline-flex min-h-9 items-center rounded-control px-3 font-ui text-meta font-medium text-mute outline-offset-2 hover:text-ink"
                onClick={() => navigate({})}
                type="button"
              >
                Clear all
              </button>
            </li>
          ) : null}
        </ul>
      ) : null}

      <div className="mt-6">
        <RefinePanel
          account={account}
          filters={filters}
          id={panel}
          navigate={navigate}
          open={refining}
          overview={overview}
          preferenceError={preferenceError}
          savingPreference={savingPreference}
          setPreference={(next) => void setPreference(next)}
          visibility={visibility}
        />
      </div>

      <div aria-busy={pending || undefined} className="mt-8 pb-chapter">
        {suppressionKey && suppressionKey !== dismissedSuppression ? (
          <output className="mb-8 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-plate bg-deep px-5 py-4">
            <span className="font-ui text-ui text-ink">
              {overview.suppressed === 1
                ? "1 matching creation is hidden by your adult-content preference."
                : `${overview.suppressed} matching creations are hidden by your adult-content preference.`}
            </span>
            <button
              className="min-h-11 font-ui text-ui font-medium text-accent underline-offset-4 hover:underline"
              onClick={openContentSetting}
              type="button"
            >
              Review the setting
            </button>
            <button
              aria-label="Dismiss the hidden-results notice"
              className="ml-auto grid size-11 place-items-center rounded-control text-mute hover:text-ink"
              onClick={() => setDismissedSuppression(suppressionKey)}
              type="button"
            >
              <X aria-hidden="true" className="size-4" />
            </button>
          </output>
        ) : null}

        {query.isPending ? <CatalogLoading /> : null}

        {query.isError ? (
          <Message
            action={
              <Button onClick={() => void query.refetch()} variant="primary">
                Try again
              </Button>
            }
            body="Check your connection, then try again."
            title="The catalog could not load."
          />
        ) : null}

        {!query.isPending && !query.isError && overview?.total === 0 ? (
          <Nothing
            clear={() => navigate({})}
            creator={creator}
            show={() => void setPreference("shown")}
            state={overview.emptyState}
            suppressed={overview.suppressed}
          />
        ) : null}

        {assets.length ? (
          <ul className={GRID}>
            {assets.map((asset, index) => (
              <CatalogPoster
                asset={asset}
                eager={index < 5}
                key={asset.id}
                visibility={visibility}
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
    </Shell>
  );
}

function CatalogLoading() {
  return (
    <output aria-label="Loading the catalog" className="block">
      <div className={GRID}>
        {[0, 1, 2, 3, 4, 5, 6, 7, 8, 9].map((slot) => (
          <div key={slot}>
            <div className="aspect-5/6 animate-pulse rounded-plate bg-deep motion-reduce:animate-none" />
            <div className="mt-4 h-4 w-3/4 animate-pulse rounded-control bg-deep motion-reduce:animate-none" />
            <div className="mt-2 h-3 w-1/2 animate-pulse rounded-control bg-deep motion-reduce:animate-none" />
          </div>
        ))}
      </div>
    </output>
  );
}

function Message({
  action,
  body,
  title,
}: {
  action?: React.ReactNode;
  body: string;
  title: string;
}) {
  return (
    <div className="rounded-plate bg-deep px-6 py-14 text-center sm:px-12">
      <h3 className="font-display text-title font-medium tracking-[-0.02em]">
        {title}
      </h3>
      <p className="mx-auto mt-3 max-w-[46ch] text-prose text-mute">{body}</p>
      {action ? <div className="mt-6 flex justify-center">{action}</div> : null}
    </div>
  );
}

function Nothing({
  clear,
  creator,
  show,
  state,
  suppressed,
}: {
  clear: () => void;
  creator?: string;
  show: () => void;
  state: BrowsePage["emptyState"];
  suppressed: number;
}) {
  if (state === "catalog") {
    return creator ? (
      <Message
        body="Nothing they have published is public yet. Their work will appear here when it is."
        title="Nothing published yet."
      />
    ) : (
      <Message
        body="The catalog will show new work here as it is published."
        title="Nothing has been published yet."
      />
    );
  }

  if (state === "suppressed") {
    return (
      <Message
        action={
          <Button onClick={show} variant="primary">
            Show adult content
          </Button>
        }
        body={
          suppressed === 1
            ? "One creation matches, and it is outside your content preference."
            : `${suppressed} creations match, and they are outside your content preference.`
        }
        title="Matching work is hidden."
      />
    );
  }

  return (
    <Message
      action={
        <Button onClick={clear} variant="primary">
          Clear the filters
        </Button>
      }
      body="Nothing matches every filter at once. Broaden the search, or start again from everything."
      title="No matches."
    />
  );
}
