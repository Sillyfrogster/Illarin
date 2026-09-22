"use client";

import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { useEffect, useId, useMemo, useState } from "react";
import { Shell } from "@/components/layout/Shell";
import { ANY_APP } from "@/components/preferences/PreferenceChoices";
import { Button } from "@/components/ui/button";
import { WorkOwnerMenu } from "@/components/work/WorkOwnerMenu";
import {
  type BrowseCursor,
  type BrowseFilters,
  type BrowsePage,
  fetchWorks,
  type NsfwPreference,
  saveAppPreference,
  saveNsfwPreference,
  workKeys,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import {
  readSessionPreference,
  writeSessionPreference,
} from "@/lib/nsfw-preference";
import { BrowsePoster } from "./BrowsePoster";
import { BrowseSearch } from "./BrowseSearch";
import { BrowseLoading, GRID, Message } from "./BrowseStates";
import { FeatureRow } from "./FeatureRow";
import { ReaderLine } from "./ReaderLine";
import { TypeIndex } from "./TypeIndex";
import { useBrowseNavigation } from "./use-browse-navigation";

const PAGE = 24;

export function BrowseSurface({
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
  search: { hint?: string; label: string; placeholder: string };
}) {
  const queryClient = useQueryClient();
  const { account } = useAuth();
  const { navigate, pending } = useBrowseNavigation(basePath);
  const panel = useId();

  const [preferenceOverride, setPreferenceOverride] =
    useState<NsfwPreference>();
  const [trouble, setTrouble] = useState("");
  const [saving, setSaving] = useState(false);
  const [answered, setAnswered] = useState(false);
  const [adultOpen, setAdultOpen] = useState(false);
  const [dismissedSuppression, setDismissedSuppression] = useState<string>();

  useEffect(() => {
    if (account !== null) {
      setPreferenceOverride(undefined);
      return;
    }
    setPreferenceOverride(readSessionPreference());
  }, [account]);

  const query = useInfiniteQuery({
    queryKey: workKeys.list(filters, preferenceOverride, creator),
    queryFn: ({ pageParam }) =>
      fetchWorks({
        ...filters,
        creator,
        limit: PAGE,
        nsfw: preferenceOverride,
        before: pageParam?.before,
        beforeId: pageParam?.beforeId,
      }),
    initialPageParam: null as BrowseCursor | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    initialData:
      initialPage && preferenceOverride === undefined
        ? { pages: [initialPage], pageParams: [null] }
        : undefined,
  });

  const pages = query.data?.pages;
  const overview = pages?.[0];
  const works = useMemo(
    () => pages?.flatMap((page) => page.items) ?? [],
    [pages],
  );
  const preference =
    preferenceOverride ?? overview?.nsfwPreference ?? "blurred";
  const narrowedTo = overview?.apps.find((app) => app.value === overview.app);
  const appLabels = useMemo(
    () => new Map(overview?.apps.map((app) => [app.value, app.label])),
    [overview?.apps],
  );
  const asking = !creator && overview?.app === null && !answered;
  const suppressionKey =
    overview?.nsfwPreference === "hidden" && overview.suppressed > 0
      ? `${JSON.stringify(filters)}:${overview.suppressed}`
      : undefined;

  async function save(write: () => Promise<void>, failed: string) {
    if (account === undefined || saving) return false;
    setTrouble("");
    setSaving(true);
    try {
      await write();
      return true;
    } catch {
      setTrouble(failed);
      return false;
    } finally {
      setSaving(false);
    }
  }

  async function setApp(app: string) {
    setAnswered(true);
    const saved = await save(async () => {
      await saveAppPreference(app, Boolean(account));
      await queryClient.invalidateQueries({ queryKey: workKeys.all });
    }, "Your app could not be saved. Try again.");
    if (!saved) setAnswered(false);
  }

  async function setPreference(next: NsfwPreference) {
    await save(async () => {
      if (account) {
        await saveNsfwPreference(next);
        await queryClient.invalidateQueries({ queryKey: workKeys.all });
      } else {
        writeSessionPreference(next);
        setPreferenceOverride(next);
      }
    }, "That preference could not be saved. Try again.");
  }

  return (
    <Shell
      aria-labelledby={`${panel}-heading`}
      as="section"
      className={cn("pb-chapter", creator ? "pt-4" : "pt-6 lg:pt-10")}
    >
      <h2 className="sr-only" id={`${panel}-heading`}>
        {heading}
      </h2>

      <div className="grid gap-y-4 md:grid-cols-[minmax(0,1fr)_minmax(15rem,21rem)] md:gap-x-10">
        <div className="min-w-0 md:col-span-2">
          <TypeIndex
            basePath={basePath}
            compact={Boolean(creator)}
            filters={filters}
            navigate={navigate}
            overview={overview}
          />
        </div>
        <div className="min-w-0 md:col-start-1 md:row-start-2 md:self-center">
          <ReaderLine
            adultOpen={adultOpen}
            asking={asking}
            locked={account === undefined || saving}
            overview={overview}
            preference={preference}
            setAdultOpen={setAdultOpen}
            setApp={creator ? null : (app) => void setApp(app)}
            setPreference={(next) => void setPreference(next)}
            signedIn={Boolean(account)}
          />
          {trouble ? (
            <p className="mt-1 font-ui text-meta text-stop" role="alert">
              {trouble}
            </p>
          ) : null}
        </div>
        <div className="min-w-0 md:col-start-2 md:row-start-2 md:self-center">
          <BrowseSearch
            hint={search.hint}
            id={`${panel}-search`}
            label={search.label}
            onSearch={(q) => navigate({ ...filters, q })}
            placeholder={search.placeholder}
            value={filters.q ?? ""}
          />
        </div>
        {overview?.facets.length ? (
          <div className="min-w-0 md:col-span-2">
            <FeatureRow
              facets={overview.facets}
              filters={filters}
              navigate={navigate}
            />
          </div>
        ) : null}
      </div>

      <div
        aria-busy={pending || query.isFetching || undefined}
        className={cn(
          "mt-10 transition-opacity duration-200 motion-reduce:transition-none",
          (pending || (query.isFetching && !query.isFetchingNextPage)) &&
            "opacity-60",
        )}
      >
        {suppressionKey && suppressionKey !== dismissedSuppression ? (
          <output className="mb-8 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-plate bg-deep px-5 py-4">
            <span className="font-ui text-ui text-ink">
              {overview.suppressed === 1
                ? "1 matching work is hidden by your adult content setting."
                : `${overview.suppressed} matching works are hidden by your adult content setting.`}
            </span>
            <button
              className="min-h-11 font-ui text-ui font-medium text-accent underline-offset-4 hover:underline"
              onClick={() => setAdultOpen(true)}
              type="button"
            >
              Change adult content
            </button>
            <button
              aria-label="Dismiss"
              className="ml-auto grid size-11 place-items-center rounded-control text-mute hover:text-ink"
              onClick={() => setDismissedSuppression(suppressionKey)}
              type="button"
            >
              <X aria-hidden="true" className="size-4" />
            </button>
          </output>
        ) : null}

        {query.isPending ? <BrowseLoading /> : null}

        {query.isError ? (
          <Message
            action={
              <Button onClick={() => void query.refetch()} variant="primary">
                Try again
              </Button>
            }
            body="Check your connection, then try again."
            title="Browse could not load"
          />
        ) : null}

        {!query.isPending && !query.isError && overview?.total === 0 ? (
          <Nothing
            app={
              narrowedTo &&
              !filters.type &&
              !filters.q &&
              !filters.facet?.length
                ? narrowedTo.label
                : undefined
            }
            clear={() => navigate({})}
            creator={creator}
            everything={() => void setApp(ANY_APP)}
            show={() => void setPreference("shown")}
            state={overview.emptyState}
            suppressed={overview.suppressed}
          />
        ) : null}

        {works.length ? (
          <ul className={GRID}>
            {works.map((work, index) => (
              <BrowsePoster
                action={
                  basePath === "/work" && account?.handle === creator ? (
                    <WorkOwnerMenu work={work} />
                  ) : undefined
                }
                apps={
                  narrowedTo
                    ? undefined
                    : work.apps.map((app) => appLabels.get(app) ?? app)
                }
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
    </Shell>
  );
}

function Nothing({
  app,
  clear,
  creator,
  everything,
  show,
  state,
  suppressed,
}: {
  app?: string;
  clear: () => void;
  creator?: string;
  everything: () => void;
  show: () => void;
  state: BrowsePage["emptyState"];
  suppressed: number;
}) {
  if (state === "nothing_published") {
    return creator ? (
      <Message title="Nothing published yet" />
    ) : (
      <Message
        body="Upload a character or a lorebook and it shows up here."
        title="Nothing published yet"
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
            ? "1 matching work is hidden by your adult content setting."
            : `${suppressed} matching works are hidden by your adult content setting.`
        }
        title="Matching works are hidden"
      />
    );
  }

  if (app) {
    return (
      <Message
        action={
          <Button onClick={everything} variant="primary">
            Show everything
          </Button>
        }
        title={`Nothing for ${app} yet`}
      />
    );
  }

  return (
    <Message
      action={
        <Button onClick={clear} variant="primary">
          Clear filters
        </Button>
      }
      body="Try a different search or remove filters."
      title="No matches"
    />
  );
}
