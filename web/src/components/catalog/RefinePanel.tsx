"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
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

export function RefinePanel({
  account,
  filters,
  id,
  navigate,
  open,
  overview,
  preferenceError,
  savingPreference,
  setPreference,
  visibility,
}: {
  account: unknown;
  filters: BrowseFilters;
  id: string;
  navigate: (next: BrowseFilters) => void;
  open: boolean;
  overview: BrowsePage | undefined;
  preferenceError: string;
  savingPreference: boolean;
  setPreference: (next: NsfwVisibility) => void;
  visibility: NsfwVisibility;
}) {
  const still = useReducedMotion();
  const apps = overview?.platforms ?? [];
  const facets = overview?.facets ?? [];

  function toggleFacet(key: string, value: string, selected: boolean) {
    const encoded = `${key}=${value}`;
    const kept = (filters.facet ?? []).filter((facet) => facet !== encoded);
    if (!selected) kept.push(encoded);
    navigate({ ...filters, facet: kept.length ? kept : undefined });
  }

  return (
    <AnimatePresence initial={false}>
      {open ? (
        <motion.div
          animate={{ height: "auto", opacity: 1 }}
          className="overflow-hidden"
          exit={{ height: 0, opacity: 0 }}
          id={id}
          initial={{ height: 0, opacity: 0 }}
          transition={still ? { duration: 0 } : MORPH}
        >
          <div className="grid gap-x-12 gap-y-8 border-t border-rule pt-8 pb-2 md:grid-cols-2 xl:grid-cols-3">
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
                    onChange={() =>
                      navigate({ ...filters, platform: app.value })
                    }
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
        </motion.div>
      ) : null}
    </AnimatePresence>
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
