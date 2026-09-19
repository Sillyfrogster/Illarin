"use client";

import { Hash, Webhook } from "lucide-react";
import type { ReactNode } from "react";
import type {
  BlogAnnouncementType,
  BlogIntegration,
  BlogIntegrationType,
  PublicationCategory,
} from "@/lib/api/query";
import { ANNOUNCEMENT_WORDS, EVENTS } from "@/lib/blog-announcement-attempt";
import { cn } from "@/lib/cn";

const BOX =
  "size-5 shrink-0 appearance-none rounded-[5px] bg-deep outline-offset-3 inset-ring-1 inset-ring-rule checked:bg-action checked:inset-ring-0 checked:bg-[length:14px] checked:bg-center checked:bg-no-repeat disabled:opacity-40";

const TICK =
  "url(\"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16' fill='none' stroke='white' stroke-width='2.4' stroke-linecap='round' stroke-linejoin='round'><path d='M3 8.5 6.5 12 13 4.5'/></svg>\")";

const DOT =
  "url(\"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'><circle cx='8' cy='8' r='4' fill='white'/></svg>\")";

export function Choice({
  children,
  hint,
  legend,
}: {
  children: ReactNode;
  hint?: string;
  legend: string;
}) {
  return (
    <fieldset className="min-w-0">
      <legend className="font-ui text-ui text-ink">{legend}</legend>
      {hint ? (
        <p className="mt-1 max-w-[52ch] font-ui text-meta text-mute">{hint}</p>
      ) : null}
      <div className="mt-3">{children}</div>
    </fieldset>
  );
}

export function Allow({
  checked,
  disabled,
  name,
  onChange,
  radio,
  slug,
  under,
  word,
}: {
  checked: boolean;
  disabled?: boolean;
  name?: string;
  onChange: (on: boolean) => void;
  radio?: boolean;
  slug?: string;
  under?: string;
  word: string;
}) {
  return (
    <label
      className={cn(
        "flex min-h-11 cursor-pointer gap-3 rounded-control px-2.5 py-1.5 hover:bg-deep",
        under ? "items-start py-2.5" : "items-center",
        disabled && "cursor-not-allowed opacity-45 hover:bg-transparent",
      )}
    >
      <input
        checked={checked}
        className={cn(BOX, under && "mt-0.5", radio && "rounded-full")}
        disabled={disabled}
        name={name}
        onChange={(event) => onChange(event.target.checked)}
        style={{ backgroundImage: checked ? (radio ? DOT : TICK) : undefined }}
        type={radio ? "radio" : "checkbox"}
      />
      <span className="min-w-0">
        <span className="block font-ui text-ui text-ink wrap-anywhere">
          {word}
          {slug ? (
            <span className="ml-2 font-mono text-label text-mute">{slug}</span>
          ) : null}
        </span>
        {under ? (
          <span className="block font-prose text-meta text-mute">{under}</span>
        ) : null}
      </span>
    </label>
  );
}

function Fallback({
  checked,
  disabled,
  name,
  onChange,
  radio,
}: {
  checked: boolean;
  disabled: boolean;
  name: string;
  onChange: (on: boolean) => void;
  radio?: boolean;
}) {
  return (
    <label
      className={cn(
        "flex min-h-11 shrink-0 cursor-pointer items-center gap-2 rounded-control px-2.5 font-ui text-meta",
        disabled
          ? "cursor-not-allowed text-mute opacity-45"
          : checked
            ? "text-accent"
            : "text-mute hover:text-ink",
      )}
    >
      <input
        checked={checked}
        className={cn(BOX, "size-4", radio && "rounded-full")}
        disabled={disabled}
        name={radio ? name : undefined}
        onChange={(event) => onChange(event.target.checked)}
        style={{ backgroundImage: checked ? (radio ? DOT : TICK) : undefined }}
        type={radio ? "radio" : "checkbox"}
      />
      Default
    </label>
  );
}

export function CategoryChoice({
  allowed,
  categories,
  fallback,
  name,
  onAllowed,
  onFallback,
}: {
  allowed: string[];
  categories: PublicationCategory[];
  fallback: string;
  name: string;
  onAllowed: (allowed: string[]) => void;
  onFallback: (fallback: string) => void;
}) {
  const offered = categories.filter((one) => !one.retired);

  function toggle(id: string, on: boolean) {
    const next = on ? [...allowed, id] : allowed.filter((held) => held !== id);
    onAllowed(next);
    if (!on && fallback === id) onFallback(next[0] ?? "");
    if (on && !fallback) onFallback(id);
  }

  return (
    <Choice legend="Allowed categories">
      <ul className="flex list-none flex-col">
        {offered.map((one) => {
          const on = allowed.includes(one.id);
          return (
            <li
              className="flex flex-wrap items-center justify-between gap-x-3"
              key={one.id}
            >
              <Allow
                checked={on}
                onChange={(next) => toggle(one.id, next)}
                slug={one.slug}
                word={one.label}
              />
              <Fallback
                checked={fallback === one.id}
                disabled={!on}
                name={`${name}-default-category`}
                onChange={() => onFallback(one.id)}
                radio
              />
            </li>
          );
        })}
      </ul>
    </Choice>
  );
}

export function IntegrationChoice({
  allowed,
  defaults,
  integrations,
  inherit,
  legend,
  onAllowed,
  onDefaults,
  onInherit,
}: {
  allowed: string[];
  defaults: string[];
  integrations: BlogIntegration[];
  inherit?: { label: string; on: boolean };
  legend: string;
  onAllowed: (allowed: string[]) => void;
  onDefaults: (defaults: string[]) => void;
  onInherit?: (on: boolean) => void;
}) {
  const offered = integrations.filter((one) => one.state !== "disabled");

  function toggle(id: string, on: boolean) {
    onAllowed(on ? [...allowed, id] : allowed.filter((held) => held !== id));
    if (!on) onDefaults(defaults.filter((held) => held !== id));
  }

  return (
    <Choice legend={legend}>
      {inherit && onInherit ? (
        <Allow checked={inherit.on} onChange={onInherit} word={inherit.label} />
      ) : null}
      {inherit?.on ? null : offered.length === 0 ? (
        <p className="font-prose text-meta text-mute">
          Nothing is set up to receive an announcement yet.
        </p>
      ) : (
        <ul className="flex list-none flex-col">
          {offered.map((one) => {
            const on = allowed.includes(one.id);
            return (
              <li
                className="flex flex-wrap items-center justify-between gap-x-3"
                key={one.id}
              >
                <Allow
                  checked={on}
                  onChange={(next) => toggle(one.id, next)}
                  under={one.host}
                  word={one.name}
                />
                <Fallback
                  checked={defaults.includes(one.id)}
                  disabled={!on}
                  name={`${one.id}-default-integration`}
                  onChange={(next) =>
                    onDefaults(
                      next
                        ? [...defaults, one.id]
                        : defaults.filter((held) => held !== one.id),
                    )
                  }
                />
              </li>
            );
          })}
        </ul>
      )}
    </Choice>
  );
}

export function AnnouncementChoice({
  chosen,
  onChosen,
}: {
  chosen: BlogAnnouncementType[];
  onChosen: (chosen: BlogAnnouncementType[]) => void;
}) {
  function toggle(event: BlogAnnouncementType, on: boolean) {
    onChosen(
      EVENTS.filter((one) => (one === event ? on : chosen.includes(one))),
    );
  }

  return (
    <Choice legend="Announcements it receives">
      <ul className="flex list-none flex-col">
        {EVENTS.map((one) => (
          <li key={one}>
            <Allow
              checked={chosen.includes(one)}
              onChange={(on) => toggle(one, on)}
              under={ANNOUNCEMENT_WORDS[one].what}
              word={ANNOUNCEMENT_WORDS[one].word}
            />
          </li>
        ))}
      </ul>
      {chosen.length === 0 ? (
        <p className="mt-2 max-w-[52ch] font-prose text-meta text-stop">
          Choose at least one announcement, or switch the integration off.
        </p>
      ) : null}
    </Choice>
  );
}

const TYPES: {
  icon: typeof Hash;
  kind: BlogIntegrationType;
  what: string;
  word: string;
}[] = [
  {
    icon: Hash,
    kind: "discord",
    what: "One channel. Illarin writes the announcement and sends it once, when a post first goes live.",
    word: "Discord",
  },
  {
    icon: Webhook,
    kind: "webhook",
    what: "Your endpoint receives signed summaries of the announcements you choose.",
    word: "Webhook",
  },
];

export function IntegrationKind({
  chosen,
  onChosen,
}: {
  chosen: BlogIntegrationType;
  onChosen: (kind: BlogIntegrationType) => void;
}) {
  return (
    <Choice legend="Integration type">
      <div className="flex flex-col gap-2">
        {TYPES.map((one) => (
          <label
            className={cn(
              "flex cursor-pointer flex-col gap-2 rounded-plate p-4 outline-offset-3 transition-colors duration-200 motion-reduce:transition-none",
              one.kind === chosen
                ? "bg-accent-wash inset-ring-2 inset-ring-accent"
                : "bg-deep hover:bg-rule/45",
            )}
            key={one.kind}
          >
            <span className="flex items-center gap-2">
              <input
                checked={one.kind === chosen}
                className="sr-only"
                name="integration-kind"
                onChange={() => onChosen(one.kind)}
                type="radio"
              />
              <one.icon
                aria-hidden="true"
                className={cn(
                  "size-4",
                  one.kind === chosen ? "text-accent" : "text-mute",
                )}
                strokeWidth={1.9}
              />
              <span className="font-ui text-ui font-medium text-ink">
                {one.word}
              </span>
            </span>
            <span className="font-prose text-meta text-mute">{one.what}</span>
          </label>
        ))}
      </div>
    </Choice>
  );
}
