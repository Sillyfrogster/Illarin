"use client";

import { AnimatePresence } from "framer-motion";
import {
  ChevronRight,
  CircleCheck,
  CircleDashed,
  CircleSlash,
  Hash,
  Plus,
  RefreshCw,
  SatelliteDish,
  Webhook,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Mark, type Tone } from "@/components/register/RowParts";
import { Button } from "@/components/ui/button";
import { Gate } from "@/components/ui/gate";
import { WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import {
  readUpdateDestinations,
  type WorkUpdateDestination,
} from "@/lib/api/work-destinations";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import {
  destinationRotating,
  destinationStanding,
  destinationWhere,
} from "@/lib/update-destinations";
import { DestinationEditor } from "./DestinationEditor";

const STATES: Record<
  WorkUpdateDestination["state"],
  { icon: typeof CircleCheck; tone: Tone; word: string }
> = {
  active: { icon: CircleCheck, tone: "accent", word: "Ready" },
  disabled: { icon: CircleSlash, tone: "quiet", word: "Switched off" },
  unverified: { icon: CircleDashed, tone: "quiet", word: "Not verified" },
};

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
        line="Sign in to manage where updates to your work are announced."
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
    WorkUpdateDestination[] | null
  >(null);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<{
    id: string | null;
    key: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

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

  function saved(destination: WorkUpdateDestination) {
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
  }

  const held = destinations ?? [];

  return (
    <>
      <div
        className={cn(
          "mt-12 max-w-[54rem] min-w-0 transition-[padding] duration-500 ease-wipe motion-reduce:transition-none",
          editing && "lg:max-w-none lg:pr-[30rem]",
        )}
      >
        <section aria-labelledby="your-destinations">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <h2
              className="font-display text-section font-medium tracking-tight text-ink"
              id="your-destinations"
            >
              Your destinations
            </h2>
            <Button
              disabled={busy}
              onClick={() => setEditing({ id: null, key: crypto.randomUUID() })}
              variant="primary"
            >
              <Plus aria-hidden="true" className="size-4" strokeWidth={2} />
              Connect a destination
            </Button>
          </div>

          <div className="mt-6">
            {error ? (
              <div className="flex flex-wrap items-center gap-4 rounded-plate bg-stop-wash px-5 py-4">
                <p className="min-w-0 flex-1 basis-56 font-ui text-ui text-stop">
                  {error}
                </p>
                <Button onClick={() => void load()} variant="secondary">
                  Try again
                </Button>
              </div>
            ) : destinations === null ? (
              <output className="block rounded-plate bg-deep px-6 py-6 text-ui text-mute">
                Loading your destinations…
              </output>
            ) : held.length === 0 ? (
              <div className="flex flex-wrap items-center gap-4 rounded-plate bg-deep px-6 py-6">
                <SatelliteDish
                  aria-hidden="true"
                  className="size-6 shrink-0 text-mute"
                  strokeWidth={1.5}
                />
                <p className="min-w-0 flex-1 basis-64 font-prose text-ui text-mute">
                  No destinations connected. Add a Discord channel or a webhook
                  you run, then select it when you publish an update.
                </p>
              </div>
            ) : (
              <ul className="m-0 grid list-none gap-2 p-0">
                {held.map((one) => (
                  <DestinationRow
                    key={one.id}
                    onOpen={() =>
                      setEditing({ id: one.id, key: `${one.id}-open` })
                    }
                    one={one}
                    open={editing?.id === one.id}
                  />
                ))}
              </ul>
            )}
          </div>
        </section>

        <ReceiverNotes />
      </div>

      <AnimatePresence>
        {editing !== null ? (
          <WorkspaceRail
            description={
              editing.id
                ? "Changes take effect on the next update you publish."
                : "Connecting a destination sends no announcement. Select it when you publish an update."
            }
            key={editing.key}
            onClose={close}
            title={editing.id ? "Destination" : "Connect a destination"}
          >
            <DestinationEditor
              busy={busy}
              existing={held.find((one) => one.id === editing.id) ?? null}
              onBusy={setBusy}
              onRemoved={(id) => {
                setDestinations(
                  (current) => current?.filter((one) => one.id !== id) ?? [],
                );
                setEditing(null);
              }}
              onSaved={saved}
            />
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>
    </>
  );
}

function DestinationRow({
  one,
  onOpen,
  open,
}: {
  one: WorkUpdateDestination;
  onOpen: () => void;
  open: boolean;
}) {
  const state = STATES[one.state];
  const Icon = one.type === "discord" ? Hash : Webhook;

  return (
    <li
      className={cn(
        "group relative rounded-plate px-5 py-5 transition-colors duration-200 motion-reduce:transition-none",
        open ? "bg-rule/45" : "bg-deep hover:bg-rule/45",
      )}
    >
      <div className="flex flex-wrap items-start gap-x-5 gap-y-4">
        <span
          className={cn(
            "grid size-11 shrink-0 place-items-center rounded-control",
            one.state === "active"
              ? "bg-accent-wash text-accent"
              : "bg-plane text-mute",
          )}
        >
          <Icon aria-hidden="true" className="size-5" strokeWidth={1.6} />
        </span>

        <div className="min-w-0 flex-1 basis-64">
          <h3 className="font-ui text-ui font-medium text-ink wrap-anywhere">
            <button
              className="text-left outline-offset-3 before:absolute before:inset-0 before:content-[''] hover:text-accent"
              onClick={onOpen}
              type="button"
            >
              {one.name}
              <span className="sr-only">. Open this destination</span>
            </button>
          </h3>
          <p className="font-ui text-ui text-mute wrap-anywhere">
            {destinationWhere(one)}
          </p>
          <p className="mt-2 max-w-[54ch] font-prose text-meta text-mute">
            {destinationStanding(one)}
          </p>
        </div>

        <div className="relative flex shrink-0 items-center gap-3">
          {destinationRotating(one) ? (
            <Mark icon={RefreshCw}>Rotating</Mark>
          ) : null}
          <Mark icon={state.icon} tone={state.tone}>
            {state.word}
          </Mark>
          <ChevronRight
            aria-hidden="true"
            className="size-4 text-mute transition-transform duration-200 group-hover:translate-x-0.5 motion-reduce:transition-none"
          />
        </div>
      </div>
    </li>
  );
}

const NOTES: { said: string; title: string }[] = [
  {
    said: "The name, the update number or your version label, your one-line summary and a link to the history. Never the changes themselves, your notes or prompt text.",
    title: "What an announcement carries",
  },
  {
    said: "Only a published update. A first publication, a private save and a correction to published notes send nothing, and an unlisted work stays quiet unless you say its link may travel.",
    title: "When one is sent",
  },
  {
    said: "A destination that does not answer is tried again for about three days. Every attempt carries the same webhook-id, so a receiver that has seen that value can drop the repeat.",
    title: "If a destination is down",
  },
  {
    said: "Two announcements can arrive in either order, so compare occurredAt and the update number rather than arrival order. Verify Illarin's signature with the signing secret saved on your server.",
    title: "What a receiver should check",
  },
];

function ReceiverNotes() {
  return (
    <section
      aria-labelledby="how-announcements-work"
      className="mt-14 border-t border-rule pt-8"
    >
      <h2
        className="font-display text-section font-medium tracking-tight text-ink"
        id="how-announcements-work"
      >
        How announcements behave
      </h2>
      <dl className="mt-6 grid gap-x-12 gap-y-7 sm:grid-cols-2">
        {NOTES.map((note) => (
          <div key={note.title}>
            <dt className="font-ui text-ui font-medium text-ink">
              {note.title}
            </dt>
            <dd className="mt-1.5 max-w-[46ch] font-prose text-meta text-mute">
              {note.said}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
