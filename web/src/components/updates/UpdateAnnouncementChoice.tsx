"use client";

import { Hash, Webhook } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  readWorkUpdateDestinationChoices,
  type WorkUpdateDestinationChoice,
} from "@/lib/api/work-destinations";
import { cn } from "@/lib/cn";

export type AnnouncementChoice = {
  destinationIds: string[] | null;
  announceUnlisted: boolean;
  notify: boolean;
};

export const NO_CHOICE: AnnouncementChoice = {
  destinationIds: null,
  announceUnlisted: false,
  notify: true,
};

/** A quiet choice sends nothing anywhere, with no destination and no notification. */
function isQuiet(choice: AnnouncementChoice): boolean {
  return !choice.notify && (choice.destinationIds ?? []).length === 0;
}

export function UpdateAnnouncementChoice({
  workId,
  choice,
  disabled,
  needsConsent,
  onChange,
  unlisted,
}: {
  workId: string;
  choice: AnnouncementChoice;
  disabled: boolean;
  needsConsent: boolean;
  onChange: (choice: AnnouncementChoice) => void;
  unlisted: boolean;
}) {
  const [offered, setOffered] = useState<WorkUpdateDestinationChoice[] | null>(
    null,
  );
  const [error, setError] = useState("");
  const report = useRef(onChange);
  report.current = onChange;

  const defaults = useCallback(
    (destinations: WorkUpdateDestinationChoice[]) =>
      unlisted
        ? []
        : destinations.filter((one) => one.byDefault).map((one) => one.id),
    [unlisted],
  );

  const load = useCallback(
    async (signal?: AbortSignal) => {
      setError("");
      const answer = await readWorkUpdateDestinationChoices(workId, signal);
      if (signal?.aborted) return;
      if (!answer.value) {
        setError(answer.error || "Could not read your destinations.");
        return;
      }
      setOffered(answer.value.destinations);
      report.current({
        announceUnlisted: false,
        destinationIds: defaults(answer.value.destinations),
        notify: true,
      });
    },
    [workId, defaults],
  );

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const chosen = choice.destinationIds ?? [];
  const consentWanted = unlisted && chosen.length > 0;
  const quiet = isQuiet(choice);

  function setQuiet(on: boolean) {
    onChange(
      on
        ? { announceUnlisted: false, destinationIds: [], notify: false }
        : {
            announceUnlisted: false,
            destinationIds: defaults(offered ?? []),
            notify: true,
          },
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <section className="flex flex-col gap-3">
        <h4 className="text-label font-medium text-mute">Who hears about it</h4>
        <Switch
          checked={choice.notify}
          disabled={disabled}
          hint="Anyone following it, or with it installed on a linked instance, gets a notification when the file changed."
          onChange={(on) => onChange({ ...choice, notify: on })}
        >
          Tell people following this work
        </Switch>
      </section>

      <section className="flex flex-col gap-3">
        <h4 className="text-label font-medium text-mute">Announce to</h4>

        {error ? (
          <Line>
            {error} A listed work will use its saved announcement destinations.
            An unlisted work will publish without an announcement.
          </Line>
        ) : offered === null ? (
          <output className="text-meta text-mute">
            Loading your destinations…
          </output>
        ) : offered.length === 0 ? (
          <Line>
            No active destinations are available. This update will publish
            without an announcement.
          </Line>
        ) : (
          <>
            <fieldset
              className="grid gap-1 rounded-control bg-deep p-1.5"
              disabled={disabled}
            >
              <legend className="sr-only">Destinations for this update</legend>
              {offered.map((one) => {
                const on = chosen.includes(one.id);
                const Icon = one.type === "discord" ? Hash : Webhook;
                return (
                  <label
                    className={cn(
                      "flex min-h-11 cursor-pointer items-center gap-3 rounded-control px-3 py-2 text-ui transition-colors duration-200 motion-reduce:transition-none",
                      on ? "bg-plane text-ink" : "text-mute hover:bg-rule/40",
                    )}
                    key={one.id}
                  >
                    <input
                      checked={on}
                      className="size-4 shrink-0 accent-[var(--v-action)]"
                      onChange={(event) =>
                        onChange({
                          ...choice,
                          destinationIds: event.target.checked
                            ? [...chosen, one.id]
                            : chosen.filter((id) => id !== one.id),
                        })
                      }
                      type="checkbox"
                    />
                    <Icon
                      aria-hidden="true"
                      className={cn(
                        "size-4 shrink-0",
                        on ? "text-accent" : "text-mute",
                      )}
                      strokeWidth={1.8}
                    />
                    <span className="min-w-0 wrap-anywhere">{one.name}</span>
                  </label>
                );
              })}
            </fieldset>

            {consentWanted ? (
              <Switch
                checked={choice.announceUnlisted}
                disabled={disabled}
                hint="Anyone reading the announcement can open it. Unlisted updates are not announced unless you select this."
                onChange={(on) => onChange({ ...choice, announceUnlisted: on })}
                warn={needsConsent && !choice.announceUnlisted}
              >
                Send this unlisted page’s link
              </Switch>
            ) : null}

            <Line>
              {chosen.length === 0
                ? "No announcement will be sent. The update still appears on the page and in its history."
                : "Selected destinations receive the name, update number, summary and history link. They do not receive the full notes or content changes."}
            </Line>
          </>
        )}

        <Link
          className="inline-flex min-h-11 items-center self-start text-meta text-accent underline-offset-4 hover:underline"
          href="/settings/update-destinations"
        >
          {offered?.length === 0
            ? "Connect a destination"
            : "Your destinations"}
        </Link>
      </section>

      <Switch
        checked={quiet}
        disabled={disabled}
        hint="Nothing is announced and nobody is notified. The update still appears on the page and in its history."
        onChange={setQuiet}
      >
        Publish quietly
      </Switch>
    </div>
  );
}

function Switch({
  checked,
  children,
  disabled,
  hint,
  onChange,
  warn = false,
}: {
  checked: boolean;
  children: React.ReactNode;
  disabled: boolean;
  hint: string;
  onChange: (checked: boolean) => void;
  warn?: boolean;
}) {
  return (
    <label
      className={cn(
        "flex min-h-11 cursor-pointer items-start gap-3 rounded-control p-3 text-ui text-ink",
        warn ? "bg-stop-wash" : "bg-deep",
      )}
    >
      <input
        checked={checked}
        className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      <span className="min-w-0">
        {children}
        <span className="mt-1 block text-meta text-mute">{hint}</span>
      </span>
    </label>
  );
}

function Line({ children }: { children: React.ReactNode }) {
  return <p className="max-w-[52ch] text-meta text-mute">{children}</p>;
}
