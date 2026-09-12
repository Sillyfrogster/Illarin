"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerTitle,
} from "@/components/ui/drawer";
import type {
  BrowseFilters,
  BrowsePage,
  NsfwVisibility,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";

const MORPH = { type: "spring", stiffness: 500, damping: 40 } as const;

const VISIBILITY: { value: NsfwVisibility; label: string; note: string }[] = [
  { value: "hidden", label: "Hide", note: "Adult work stays out of results." },
  { value: "blurred", label: "Blur", note: "Blur adult cover images." },
  { value: "shown", label: "Show", note: "Show adult cover images." },
];

export type RefineChoices = {
  account: unknown;
  filters: BrowseFilters;
  id: string;
  navigate: (next: BrowseFilters) => void;
  overview: BrowsePage | undefined;
  preferenceError: string;
  savingPreference: boolean;
  setPreference: (next: NsfwVisibility) => void;
  visibility: NsfwVisibility;
};

/** Opens the filters in place, above the results. */
export function RefinePanel({
  open,
  ...choices
}: RefineChoices & { open: boolean }) {
  const still = useReducedMotion();

  return (
    <AnimatePresence initial={false}>
      {open ? (
        <motion.div
          animate={{ height: "auto", opacity: 1 }}
          className="overflow-hidden"
          exit={{ height: 0, opacity: 0 }}
          id={choices.id}
          initial={{ height: 0, opacity: 0 }}
          transition={still ? { duration: 0 } : MORPH}
        >
          <RefineGroups
            {...choices}
            className="border-t border-rule pt-8 pb-2 md:grid-cols-2 xl:grid-cols-3"
          />
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}

/** Holds the same filters in a drawer that rises over the results on a phone. */
export function RefineDrawer({
  clear,
  onOpenChange,
  open,
  pending,
  total,
  ...choices
}: RefineChoices & {
  clear: (() => void) | null;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  pending: boolean;
  total: number | undefined;
}) {
  return (
    <Drawer onOpenChange={onOpenChange} open={open}>
      <DrawerContent aria-describedby={undefined} id={choices.id}>
        <div className="flex items-center justify-between gap-4 pt-1 pr-2 pl-5">
          <DrawerTitle className="font-display text-section font-medium text-ink">
            Filters
          </DrawerTitle>
          <DrawerClose asChild>
            <Button aria-label="Close the filters" size="icon" variant="ghost">
              <X aria-hidden="true" />
            </Button>
          </DrawerClose>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 pt-3 pb-8">
          <RefineGroups {...choices} />
        </div>
        <div className="flex items-center gap-2 px-5 pt-3 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[0_-18px_24px_-22px_rgb(0_0_0/0.5)]">
          {clear ? (
            <Button onClick={clear} variant="ghost">
              Clear filters
            </Button>
          ) : null}
          <DrawerClose asChild>
            <Button
              aria-busy={pending || undefined}
              className="flex-1"
              variant="primary"
            >
              {resultsLabel(pending ? undefined : total)}
            </Button>
          </DrawerClose>
        </div>
      </DrawerContent>
    </Drawer>
  );
}

function resultsLabel(total: number | undefined): string {
  if (total === undefined) return "Show results";
  if (total === 0) return "No creations match";
  return total === 1 ? "Show 1 creation" : `Show ${total} creations`;
}

function RefineGroups({
  account,
  className,
  filters,
  id,
  navigate,
  overview,
  preferenceError,
  savingPreference,
  setPreference,
  visibility,
}: RefineChoices & { className?: string }) {
  const apps = overview?.platforms ?? [];
  const facets = overview?.facets ?? [];

  function toggleFacet(key: string, value: string, selected: boolean) {
    const encoded = `${key}=${value}`;
    const kept = (filters.facet ?? []).filter((facet) => facet !== encoded);
    if (!selected) kept.push(encoded);
    navigate({ ...filters, facet: kept.length ? kept : undefined });
  }

  return (
    <div className={cn("grid gap-x-12 gap-y-8", className)}>
      {apps.length ? (
        <Group legend="Works with">
          <Choice
            checked={!filters.platform}
            label="Any app"
            name={`${id}-app`}
            onChange={() => navigate({ ...filters, platform: undefined })}
            type="radio"
          />
          {apps.map((app) => (
            <Choice
              checked={filters.platform === app.value}
              count={app.count}
              disabled={app.count === 0}
              key={app.value}
              label={app.label}
              name={`${id}-app`}
              onChange={() => navigate({ ...filters, platform: app.value })}
              type="radio"
            />
          ))}
        </Group>
      ) : null}

      {facets.map((facet) => (
        <Group key={facet.key} legend={facet.label}>
          {facet.options.map((option) => (
            <Choice
              checked={option.selected}
              count={option.count}
              disabled={option.count === 0 && !option.selected}
              key={option.value}
              label={option.label}
              onChange={() =>
                toggleFacet(facet.key, option.value, option.selected)
              }
              type="checkbox"
            />
          ))}
        </Group>
      ))}

      <fieldset
        className="min-w-0 border-0 p-0"
        id={`${id}-adult`}
        tabIndex={-1}
      >
        <legend className="font-ui text-meta font-medium text-ink">
          Adult content
        </legend>
        <div className="mt-3 flex flex-wrap gap-2">
          {VISIBILITY.map((option) => (
            <button
              aria-pressed={visibility === option.value}
              className={cn(
                "min-h-11 rounded-control px-4 font-ui text-ui font-medium outline-offset-2 transition-colors duration-200 disabled:opacity-45 motion-reduce:transition-none",
                visibility === option.value
                  ? "bg-action text-on-accent"
                  : "bg-deep text-ink hover:bg-rule/45",
              )}
              disabled={account === undefined || savingPreference}
              key={option.value}
              onClick={() => setPreference(option.value)}
              type="button"
            >
              {option.label}
            </button>
          ))}
        </div>
        <p className="mt-3 max-w-[36ch] font-ui text-meta text-mute">
          {VISIBILITY.find((one) => one.value === visibility)?.note}{" "}
          {account ? "Saved to your account." : "Kept on this browser."}
        </p>
        {preferenceError ? (
          <p className="mt-2 font-ui text-meta text-stop" role="alert">
            {preferenceError}
          </p>
        ) : null}
      </fieldset>
    </div>
  );
}

function Group({
  children,
  legend,
}: {
  children: React.ReactNode;
  legend: string;
}) {
  return (
    <fieldset className="min-w-0 border-0 p-0">
      <legend className="font-ui text-meta font-medium text-ink">
        {legend}
      </legend>
      <div className="mt-1 flex flex-col">{children}</div>
    </fieldset>
  );
}

function Choice({
  checked,
  count,
  disabled,
  label,
  name,
  onChange,
  type,
}: {
  checked: boolean;
  count?: number;
  disabled?: boolean;
  label: string;
  name?: string;
  onChange: () => void;
  type: "radio" | "checkbox";
}) {
  return (
    <label
      className={cn(
        "flex min-h-11 cursor-pointer items-center gap-3 rounded-control px-2 -mx-2 font-ui text-ui text-ink transition-colors duration-200 hover:bg-deep motion-reduce:transition-none",
        disabled && "cursor-default text-mute opacity-55 hover:bg-transparent",
      )}
    >
      <input
        checked={checked}
        className="size-4 shrink-0 accent-[var(--v-action)]"
        disabled={disabled}
        name={name}
        onChange={onChange}
        type={type}
      />
      <span className="min-w-0 flex-1 [overflow-wrap:anywhere]">{label}</span>
      {count === undefined ? null : (
        <span className="shrink-0 font-ui text-meta text-mute tabular-nums">
          {count}
        </span>
      )}
    </label>
  );
}
