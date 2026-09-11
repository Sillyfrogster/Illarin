"use client";

import { Hash, Webhook } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  type AssetUpdateDestinationChoice,
  readAssetUpdateDestinationChoices,
} from "@/lib/api/asset-destinations";
import { cn } from "@/lib/cn";

export type AnnouncementChoice = {
  destinationIds: string[] | null;
  announceUnlisted: boolean;
};

export const NO_CHOICE: AnnouncementChoice = {
  destinationIds: null,
  announceUnlisted: false,
};

export function UpdateAnnouncementChoice({
  assetId,
  choice,
  disabled,
  needsConsent,
  onChange,
  unlisted,
}: {
  assetId: string;
  choice: AnnouncementChoice;
  disabled: boolean;
  needsConsent: boolean;
  onChange: (choice: AnnouncementChoice) => void;
  unlisted: boolean;
}) {
  const [offered, setOffered] = useState<AssetUpdateDestinationChoice[] | null>(
    null,
  );
  const [error, setError] = useState("");
  const report = useRef(onChange);
  report.current = onChange;

  const load = useCallback(
    async (signal?: AbortSignal) => {
      setError("");
      const answer = await readAssetUpdateDestinationChoices(assetId, signal);
      if (signal?.aborted) return;
      if (!answer.value) {
        setError(answer.error || "Could not read your destinations.");
        return;
      }
      setOffered(answer.value.destinations);
      report.current({
        announceUnlisted: false,
        destinationIds: unlisted
          ? []
          : answer.value.destinations
              .filter((one) => one.byDefault)
              .map((one) => one.id),
      });
    },
    [assetId, unlisted],
  );

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const chosen = choice.destinationIds ?? [];
  const consentWanted = unlisted && chosen.length > 0;

  return (
    <section className="flex flex-col gap-3">
      <h4 className="text-label font-medium text-mute">Announce to</h4>

      {error ? (
        <Line>
          {error} Publishing still works, and this update goes to whatever the
          asset already remembers.
        </Line>
      ) : offered === null ? (
        <output className="text-meta text-mute">
          Reading your destinations…
        </output>
      ) : offered.length === 0 ? (
        <Line>
          You have no destination connected, so this update announces nowhere.
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
              const Kind = one.kind === "discord" ? Hash : Webhook;
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
                  <Kind
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
            <label
              className={cn(
                "flex min-h-11 cursor-pointer items-start gap-3 rounded-control p-3 text-ui text-ink",
                needsConsent && !choice.announceUnlisted
                  ? "bg-stop-wash"
                  : "bg-deep",
              )}
            >
              <input
                checked={choice.announceUnlisted}
                className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
                disabled={disabled}
                onChange={(event) =>
                  onChange({
                    ...choice,
                    announceUnlisted: event.target.checked,
                  })
                }
                type="checkbox"
              />
              <span className="min-w-0">
                Send this unlisted page’s link
                <span className="mt-1 block text-meta text-mute">
                  Anyone reading the announcement can open it. Unlisted updates
                  stay quiet unless you tick this.
                </span>
              </span>
            </label>
          ) : null}

          <Line>
            {chosen.length === 0
              ? "Nothing is announced. The update still appears on the page and in its history."
              : "They receive the name, the update number, your summary and a link to the history. Never your notes or the changes themselves."}
          </Line>
        </>
      )}

      <Link
        className="inline-flex min-h-11 items-center self-start text-meta text-accent underline-offset-4 hover:underline"
        href="/settings/update-destinations"
      >
        {offered?.length === 0 ? "Connect a destination" : "Your destinations"}
      </Link>
    </section>
  );
}

function Line({ children }: { children: React.ReactNode }) {
  return <p className="max-w-[52ch] text-meta text-mute">{children}</p>;
}
