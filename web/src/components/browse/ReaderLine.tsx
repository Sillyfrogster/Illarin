"use client";

import { ChevronDown } from "lucide-react";
import type { ReactNode } from "react";
import {
  ADULT_CHOICES,
  ANY_APP,
} from "@/components/preferences/PreferenceChoices";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { BrowsePage, NsfwPreference } from "@/lib/api/query";
import { cn } from "@/lib/cn";

const ADULT_STATES: Record<NsfwPreference, string> = {
  hidden: "Adult works hidden",
  blurred: "Adult covers blurred",
  shown: "Adult covers shown",
};

const word =
  "-mx-1.5 inline-flex min-h-11 items-center gap-1 rounded-control px-1.5 font-medium text-ink outline-offset-2 transition-colors duration-200 hover:bg-deep data-[state=open]:bg-deep disabled:opacity-55 motion-reduce:transition-none";

/** ReaderLine states the reader's app and adult content setting as words that open their choices. */
export function ReaderLine({
  adultOpen,
  asking,
  locked,
  overview,
  preference,
  setAdultOpen,
  setApp,
  setPreference,
  signedIn,
}: {
  adultOpen: boolean;
  asking: boolean;
  locked: boolean;
  overview: BrowsePage | undefined;
  preference: NsfwPreference;
  setAdultOpen: (open: boolean) => void;
  setApp: ((app: string) => void) | null;
  setPreference: (next: NsfwPreference) => void;
  signedIn: boolean;
}) {
  const app = overview?.app ?? null;
  const named = overview?.apps.find((one) => one.value === app)?.label;

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 font-ui text-ui text-mute">
      {setApp && overview ? (
        asking ? (
          <span className="flex basis-full flex-wrap items-center gap-2 sm:basis-auto">
            <span className="mr-1">Works in</span>
            {overview.apps.map((option) => (
              <button
                className="inline-flex min-h-9 items-center rounded-control bg-accent-wash px-3 font-medium text-ink outline-offset-2 transition-colors duration-200 hover:bg-action hover:text-on-accent disabled:opacity-55 motion-reduce:transition-none"
                disabled={locked}
                key={option.value}
                onClick={() => setApp(option.value)}
                type="button"
              >
                {option.label}
              </button>
            ))}
            <button
              className="inline-flex min-h-9 items-center rounded-control px-2.5 font-medium text-mute outline-offset-2 hover:bg-deep hover:text-ink disabled:opacity-55"
              disabled={locked}
              onClick={() => setApp(ANY_APP)}
              type="button"
            >
              Any app
            </button>
          </span>
        ) : (
          <Choice
            label={`Works in ${named ?? "any app"}`}
            locked={locked}
            note={
              signedIn ? "Saved to your account." : "Saved in this browser."
            }
            onChange={setApp}
            options={[{ value: ANY_APP, label: "Any app" }, ...overview.apps]}
            value={app === null ? ANY_APP : app}
          />
        )
      ) : null}

      {setApp && overview ? <Dot /> : null}

      <Choice
        label={ADULT_STATES[preference]}
        locked={locked}
        note={`Blur and Show are for 18 and over.${signedIn ? " Saved to your account." : ""}`}
        onChange={(next) => setPreference(next as NsfwPreference)}
        onOpenChange={setAdultOpen}
        open={adultOpen}
        options={ADULT_CHOICES.map((one) => ({
          value: one.value,
          label: one.label,
          detail: one.note,
        }))}
        value={preference}
      />

      {overview ? (
        <>
          <Dot />
          <output aria-live="polite" className="tabular-nums">
            {overview.total === 1 ? "1 work" : `${overview.total} works`},
            newest first
          </output>
        </>
      ) : null}
    </div>
  );
}

function Dot() {
  return (
    <span aria-hidden="true" className="text-rule max-sm:hidden">
      ·
    </span>
  );
}

function Choice({
  label,
  locked,
  note,
  onChange,
  onOpenChange,
  open,
  options,
  value,
}: {
  label: string;
  locked: boolean;
  note: string;
  onChange: (value: string) => void;
  onOpenChange?: (open: boolean) => void;
  open?: boolean;
  options: { value: string; label: string; count?: number; detail?: string }[];
  value: string;
}) {
  return (
    <DropdownMenu modal={false} onOpenChange={onOpenChange} open={open}>
      <DropdownMenuTrigger className={word} disabled={locked}>
        {label}
        <ChevronDown aria-hidden="true" className="size-4 text-mute" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="w-[min(19rem,calc(100vw-2rem))]"
      >
        <DropdownMenuRadioGroup onValueChange={onChange} value={value}>
          {options.map((option) => (
            <DropdownMenuRadioItem
              className={cn(option.detail && "py-2")}
              key={option.value}
              value={option.value}
            >
              <Option {...option} />
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <p className="px-3 pt-2 pb-1.5 font-ui text-meta text-mute">{note}</p>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Option({
  count,
  detail,
  label,
}: {
  count?: number;
  detail?: string;
  label: string;
}): ReactNode {
  return (
    <span className="flex min-w-0 flex-1 items-baseline justify-between gap-3">
      <span className="min-w-0">
        <span className="block font-medium">{label}</span>
        {detail ? (
          <span className="block text-meta text-mute">{detail}</span>
        ) : null}
      </span>
      {count === undefined ? null : (
        <span className="text-meta text-mute tabular-nums">{count}</span>
      )}
    </span>
  );
}
