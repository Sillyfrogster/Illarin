"use client";

import { animate, useReducedMotion } from "framer-motion";
import { Check, ChevronDown, Eye, EyeOff, X } from "lucide-react";
import { useEffect, useRef } from "react";
import {
  ADULT_CHOICES,
  ANY_APP,
} from "@/components/preferences/PreferenceChoices";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Scroller } from "@/components/ui/scroller";
import { Toggle } from "@/components/ui/toggle";
import type {
  BrowseFilters,
  BrowsePage,
  NsfwPreference,
} from "@/lib/api/query";
import { chooseFilter } from "@/lib/browse-url";
import { cn } from "@/lib/cn";
import { SortMenu } from "./SortMenu";

type Facet = BrowsePage["facets"][number];

const ANY = "";

const ADULT_WORDS: Record<NsfwPreference, string> = {
  hidden: "Adult hidden",
  blurred: "Adult blurred",
  shown: "Adult shown",
};

const CHIP =
  "inline-flex h-control w-auto items-center gap-1.5 bg-fill px-3 text-ui font-medium text-ink hover:bg-fill-hover hover:text-ink aria-pressed:bg-accent-wash aria-pressed:text-accent [&_.count]:text-meta [&_.count]:font-normal [&_.count]:text-mute [&_.count]:tabular-nums aria-pressed:[&_.count]:text-accent";

function isPresence(facet: Facet) {
  return facet.options.some((option) => option.value === "true");
}

/** FilterLine is Browse's one line of choices: the reader's app, the type's features, the order and the adult setting. */
export function FilterLine({
  adultOpen,
  asking,
  filters,
  locked,
  navigate,
  overview,
  preference,
  setAdultOpen,
  setApp,
  setPreference,
}: {
  adultOpen: boolean;
  asking: boolean;
  filters: BrowseFilters;
  locked: boolean;
  navigate: (next: BrowseFilters) => void;
  overview: BrowsePage | undefined;
  preference: NsfwPreference;
  setAdultOpen: (open: boolean) => void;
  setApp: ((app: string) => void) | null;
  setPreference: (next: NsfwPreference) => void;
}) {
  const facets = overview?.facets ?? [];
  const presence = facets.filter(isPresence);
  const ranged = facets.filter((facet) => !isPresence(facet));
  const inUse = facets.some((facet) =>
    facet.options.some((option) => option.selected),
  );
  const app = overview?.app ?? null;
  const appName = overview?.apps.find((one) => one.value === app)?.label;

  return (
    <div className="flex min-w-0 items-center gap-2">
      <div className="-ml-4 min-w-0 flex-1">
        <Scroller>
          <div className="flex w-max items-center gap-2 pr-6 pl-4">
            {setApp && overview ? (
              <WordMenu
                label={asking ? "Pick your app" : `For ${appName ?? "any app"}`}
                locked={locked}
                onChange={setApp}
                options={[
                  { value: ANY_APP, label: "Any app" },
                  ...overview.apps,
                ]}
                tone={asking ? "ask" : "plain"}
                value={app === null ? ANY_APP : app}
              />
            ) : null}

            {presence.length ? (
              <fieldset className="m-0 flex items-center gap-2 border-0 p-0">
                <legend className="sr-only">Has</legend>
                <span
                  aria-hidden="true"
                  className="pl-2 text-meta text-mute max-md:hidden"
                >
                  Has
                </span>
                {presence.map((facet) => {
                  const yes = facet.options.find(
                    (option) => option.value === "true",
                  );
                  if (!yes) return null;
                  return (
                    <Toggle
                      className={CHIP}
                      disabled={yes.count === 0 && !yes.selected}
                      key={facet.key}
                      onPressedChange={(pressed) =>
                        navigate(
                          chooseFilter(
                            filters,
                            facet.key,
                            pressed ? "true" : null,
                          ),
                        )
                      }
                      pressed={yes.selected}
                    >
                      {yes.selected ? <Check aria-hidden="true" /> : null}
                      {facet.label}
                      <span className="count">{yes.count}</span>
                    </Toggle>
                  );
                })}
              </fieldset>
            ) : null}

            {ranged.map((facet) => {
              const chosen = facet.options.find((option) => option.selected);
              return (
                <WordMenu
                  key={facet.key}
                  label={
                    chosen ? `${facet.label}: ${chosen.label}` : facet.label
                  }
                  locked={false}
                  onChange={(value) =>
                    navigate(
                      chooseFilter(
                        filters,
                        facet.key,
                        value === ANY ? null : value,
                      ),
                    )
                  }
                  options={[
                    { value: ANY, label: "Any number" },
                    ...facet.options.map((option) => ({
                      value: option.value,
                      label: option.label,
                      count: option.count,
                    })),
                  ]}
                  tone={chosen ? "on" : "plain"}
                  value={chosen?.value ?? ANY}
                />
              );
            })}

            {inUse ? (
              <Button
                onClick={() => navigate({ ...filters, facet: undefined })}
                variant="ghost"
              >
                <X aria-hidden="true" />
                Clear
              </Button>
            ) : null}
          </div>
        </Scroller>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        {overview ? (
          <Count className="hidden pr-2 md:block" value={overview.total} />
        ) : null}
        <SortMenu filters={filters} navigate={navigate} />
        <AdultMenu
          locked={locked}
          onOpenChange={setAdultOpen}
          open={adultOpen}
          preference={preference}
          setPreference={setPreference}
        />
      </div>
    </div>
  );
}

/** Count is the number of matching works, counting to a new total when it changes. */
export function Count({
  className,
  value,
}: {
  className?: string;
  value: number;
}) {
  const shown = useRef<HTMLSpanElement>(null);
  const was = useRef(value);
  const still = useReducedMotion();

  useEffect(() => {
    const from = was.current;
    was.current = value;
    const here = shown.current;
    if (!here || from === value || still) {
      if (here) here.textContent = value.toLocaleString("en-US");
      return;
    }
    const run = animate(from, value, {
      duration: 0.5,
      ease: [0.22, 1, 0.36, 1],
      onUpdate: (at) => {
        here.textContent = Math.round(at).toLocaleString("en-US");
      },
    });
    return () => run.stop();
  }, [value, still]);

  return (
    <output
      aria-live="polite"
      className={cn("text-ui text-mute tabular-nums", className)}
    >
      <span ref={shown}>{value.toLocaleString("en-US")}</span>
      {value === 1 ? " work" : " works"}
    </output>
  );
}

function WordMenu({
  label,
  locked,
  onChange,
  options,
  tone,
  value,
}: {
  label: string;
  locked: boolean;
  onChange: (value: string) => void;
  options: { value: string; label: string; count?: number }[];
  tone: "plain" | "on" | "ask";
  value: string;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild disabled={locked}>
        <Button
          className={cn(
            "gap-1.5",
            tone !== "plain" &&
              "bg-accent-wash text-accent hover:bg-accent-wash hover:text-accent",
          )}
        >
          {label}
          <ChevronDown aria-hidden="true" className="opacity-60" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuRadioGroup onValueChange={onChange} value={value}>
          {options.map((option) => (
            <DropdownMenuRadioItem
              disabled={option.count === 0 && option.value !== value}
              key={option.value}
              value={option.value}
            >
              <span className="flex min-w-0 flex-1 items-baseline justify-between gap-4">
                {option.label}
                {option.count === undefined ? null : (
                  <span className="text-meta text-mute tabular-nums">
                    {option.count}
                  </span>
                )}
              </span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AdultMenu({
  locked,
  onOpenChange,
  open,
  preference,
  setPreference,
}: {
  locked: boolean;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  preference: NsfwPreference;
  setPreference: (next: NsfwPreference) => void;
}) {
  const Icon = preference === "shown" ? Eye : EyeOff;
  return (
    <DropdownMenu onOpenChange={onOpenChange} open={open}>
      <DropdownMenuTrigger asChild disabled={locked}>
        <Button
          aria-label={ADULT_WORDS[preference]}
          className="max-md:w-control max-md:px-0"
        >
          <Icon aria-hidden="true" />
          <span className="max-md:hidden">{ADULT_WORDS[preference]}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-[min(18rem,calc(100vw-2rem))]"
      >
        <DropdownMenuRadioGroup
          onValueChange={(next) => setPreference(next as NsfwPreference)}
          value={preference}
        >
          {ADULT_CHOICES.map((choice) => (
            <DropdownMenuRadioItem
              className="py-2"
              key={choice.value}
              value={choice.value}
            >
              <span className="flex flex-col">
                <span className="font-medium">{choice.label}</span>
                <span className="text-meta text-mute">{choice.note}</span>
              </span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <p className="px-3 pt-2 pb-1.5 text-meta text-mute">
          Blur and Show are for 18 and over.
        </p>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
