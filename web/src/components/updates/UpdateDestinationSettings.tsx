"use client";

import { Hash, Plus, Webhook } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Mark,
  Nothing,
  Rows,
  StartAction,
} from "@/components/register/RowParts";
import { Button } from "@/components/ui/button";
import { Trouble } from "@/components/ui/field";
import { Gate } from "@/components/ui/gate";
import {
  type AssetUpdateDestination,
  readUpdateDestinations,
} from "@/lib/api/asset-destinations";
import { useAuth } from "@/lib/auth";
import { DestinationEditor } from "./DestinationEditor";

export const destinationState = {
  active: "Ready",
  unverified: "Needs verification",
  disabled: "Disabled",
} as const;

export function UpdateDestinationSettings() {
  const { account } = useAuth();
  if (account === undefined)
    return (
      <output className="mt-10 block text-ui text-mute">
        Loading your account…
      </output>
    );
  if (!account)
    return (
      <Gate
        action="Sign in"
        className="mt-10"
        heading="Your destinations"
        href="/sign-in?returnTo=%2Fsettings%2Fupdate-destinations"
        line="Sign in to manage where your asset updates are announced."
      />
    );
  if (!account.emailVerified)
    return (
      <Gate
        action="Verify email"
        className="mt-10"
        heading="Verify your account"
        href="/verify-email?returnTo=%2Fsettings%2Fupdate-destinations"
        line="Verify your email before connecting an update destination."
      />
    );
  return <DestinationManager key={account.id} />;
}

function DestinationManager() {
  const [destinations, setDestinations] = useState<
    AssetUpdateDestination[] | null
  >(null);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<{
    id: string | null;
    key: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);
  const listHeading = useRef<HTMLHeadingElement>(null);

  const load = useCallback(async (signal?: AbortSignal) => {
    setError("");
    const answer = await readUpdateDestinations(signal);
    if (signal?.aborted) return;
    if (!answer.value)
      setError(answer.error || "Could not read your destinations.");
    else setDestinations(answer.value.destinations);
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  function saved(destination: AssetUpdateDestination) {
    setEditing((current) =>
      current ? { ...current, id: destination.id } : current,
    );
    setDestinations((current) => {
      const rest = (current ?? []).filter((one) => one.id !== destination.id);
      return [...rest, destination].sort((a, b) =>
        a.name.localeCompare(b.name),
      );
    });
  }

  function close() {
    if (busy) return;
    setEditing(null);
    listHeading.current?.focus();
  }

  return (
    <div className="mt-12 grid min-w-0 gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.85fr)] lg:gap-16">
      <section aria-labelledby="your-destinations" className="min-w-0">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <h2
            className="font-display text-section font-medium tracking-tight text-ink outline-offset-3"
            id="your-destinations"
            ref={listHeading}
            tabIndex={-1}
          >
            Connected destinations
          </h2>
          <StartAction
            disabled={busy}
            icon={Plus}
            onClick={() => setEditing({ id: null, key: crypto.randomUUID() })}
          >
            Add destination
          </StartAction>
        </div>
        {error ? (
          <div className="mt-6 grid justify-items-start gap-3">
            <Trouble>{error}</Trouble>
            <Button onClick={() => void load()} variant="secondary">
              Try again
            </Button>
          </div>
        ) : destinations === null ? (
          <output className="mt-6 block text-ui text-mute">
            Loading your destinations…
          </output>
        ) : destinations.length === 0 ? (
          <Nothing>
            Connect the channel where your community follows your work, or an
            endpoint you run. New assets are always published quietly.
          </Nothing>
        ) : (
          <Rows>
            {destinations.map((destination) => {
              const Icon = destination.kind === "discord" ? Hash : Webhook;
              return (
                <li
                  className="min-w-0 border-rule not-first:border-t"
                  key={destination.id}
                >
                  <button
                    aria-pressed={editing?.id === destination.id}
                    className="flex w-full min-w-0 items-start gap-4 rounded-plate px-4 py-5 text-left outline-offset-3 hover:bg-deep disabled:opacity-50 sm:px-5"
                    disabled={busy}
                    onClick={() =>
                      setEditing({ id: destination.id, key: destination.id })
                    }
                    type="button"
                  >
                    <Icon
                      aria-hidden="true"
                      className="mt-1 size-5 shrink-0 text-accent"
                    />
                    <span className="grid min-w-0 flex-1 gap-2">
                      <span className="font-display text-section font-medium text-ink wrap-anywhere">
                        {destination.name}
                      </span>
                      <span className="text-meta text-mute wrap-anywhere">
                        {destination.kind === "discord"
                          ? "Discord channel"
                          : destination.host}
                      </span>
                      <span>
                        <Mark
                          tone={
                            destination.state === "active" ? "accent" : "quiet"
                          }
                        >
                          {destinationState[destination.state]}
                        </Mark>
                      </span>
                    </span>
                  </button>
                </li>
              );
            })}
          </Rows>
        )}
      </section>
      {editing !== null ? (
        <DestinationEditor
          busy={busy}
          existing={destinations?.find((one) => one.id === editing.id) ?? null}
          key={editing.key}
          onBusy={setBusy}
          onClose={close}
          onRemoved={(id) => {
            setDestinations(
              (current) => current?.filter((one) => one.id !== id) ?? [],
            );
            setEditing(null);
            listHeading.current?.focus();
          }}
          onSaved={saved}
        />
      ) : (
        <aside className="min-w-0 rounded-plate bg-deep p-6 lg:self-start lg:p-8">
          <h2 className="font-display text-section font-medium text-ink">
            One connection, your choice of updates
          </h2>
          <p className="mt-3 max-w-[46ch] font-prose text-ui text-mute">
            Save destinations here, then choose defaults in each asset’s
            publication workspace. Connecting a destination sends no
            announcement.
          </p>
          <p className="mt-4 max-w-[46ch] font-prose text-ui text-mute">
            Only you can manage these connections. Webhook addresses and saved
            credentials stay masked.
          </p>
        </aside>
      )}
    </div>
  );
}
