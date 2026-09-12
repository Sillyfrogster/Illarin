"use client";

import { useEffect, useState } from "react";
import { TextArea } from "@/components/ui/field";
import { readPostDestinations } from "@/lib/api/posts";
import type { PublicationDestinationChoice } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import {
  eventWord,
  offeredFor,
  type Transition,
  transitionEvent,
} from "@/lib/publication-delivery";

const QUIET: Record<Transition, string> = {
  publish:
    "Nothing is sent. The post still appears on the blog and in the feeds.",
  changes: "Nothing is sent. The changes still go live.",
  withdraw: "No announcement will be sent. The post will still be withdrawn.",
  republish:
    "No announcement will be sent. The post will still be republished.",
};

const SENT: Record<Transition, string> = {
  publish: "Each one receives a summary and a link, never the article itself.",
  changes:
    "Each destination receives an update announcement for the new revision.",
  withdraw:
    "Each destination receives a withdrawal notice without the private reason.",
  republish:
    "Each destination receives an announcement for the republished revision.",
};

export function AnnouncementChoice({
  postId,
  transition,
  announced,
  chosen,
  pinging,
  note,
  onChosen,
  onPinging,
  onNote,
}: {
  postId: string;
  transition: Transition;
  announced: boolean;
  chosen: string[] | null;
  pinging: string[];
  note: string;
  onChosen: (chosen: string[]) => void;
  onPinging: (pinging: string[]) => void;
  onNote: (note: string) => void;
}) {
  const [offered, setOffered] = useState<PublicationDestinationChoice[]>([]);

  useEffect(() => {
    let live = true;
    void readPostDestinations(postId).then((answer) => {
      if (live && answer.value) setOffered(answer.value.destinations);
    });
    return () => {
      live = false;
    };
  }, [postId]);

  const event = transitionEvent(transition);
  const takers = offered.filter((one) => offeredFor(one, event, announced));

  if (takers.length === 0) {
    if (offered.length === 0) return null;
    return (
      <p className="font-prose text-meta text-mute">
        No destination receives {eventWord(event)} announcements.
      </p>
    );
  }

  const picked =
    chosen ?? takers.filter((one) => one.byDefault).map((one) => one.id);

  function toggle(id: string, on: boolean) {
    onChosen(on ? [...picked, id] : picked.filter((held) => held !== id));
    if (!on) onPinging(pinging.filter((held) => held !== id));
  }

  function togglePing(id: string, on: boolean) {
    onPinging(on ? [...pinging, id] : pinging.filter((held) => held !== id));
  }

  return (
    <div className="flex flex-col gap-3">
      <fieldset className="flex flex-col gap-2 border-0">
        <legend className="mb-1 font-ui text-ui text-ink">
          Send announcement
        </legend>
        <ul className="flex list-none flex-col gap-1.5">
          {takers.map((one) => (
            <li className="flex flex-col gap-1.5" key={one.id}>
              <Toggle
                checked={picked.includes(one.id)}
                onChange={(on) => toggle(one.id, on)}
              >
                {one.name}
              </Toggle>
              {one.role && picked.includes(one.id) ? (
                <Toggle
                  checked={pinging.includes(one.id)}
                  className="ml-6"
                  hint="Everyone with that role gets a notification. It cannot be taken back."
                  onChange={(on) => togglePing(one.id, on)}
                >
                  Ping @{one.role}
                </Toggle>
              ) : null}
            </li>
          ))}
        </ul>
        <p className="font-prose text-meta text-mute">
          {picked.length === 0 ? QUIET[transition] : SENT[transition]}
        </p>
      </fieldset>

      {picked.length > 0 ? (
        <div className="flex flex-col gap-2">
          <label
            className="font-ui text-ui text-ink"
            htmlFor="announcement-note"
          >
            Note
          </label>
          <TextArea
            className="min-h-20"
            id="announcement-note"
            maxLength={500}
            onChange={(event) => onNote(event.target.value)}
            placeholder="Optional context included in each announcement."
            rows={2}
            value={note}
          />
          <p className="font-prose text-meta text-mute">
            It travels with the announcement and never becomes part of the post.
          </p>
        </div>
      ) : null}
    </div>
  );
}

function Toggle({
  checked,
  children,
  className,
  hint,
  onChange,
}: {
  checked: boolean;
  children: React.ReactNode;
  className?: string;
  hint?: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label
      className={cn(
        "flex min-h-11 cursor-pointer items-start gap-3 rounded-control p-3 font-ui text-ui text-ink",
        checked ? "bg-accent-wash" : "bg-deep",
        className,
      )}
    >
      <input
        checked={checked}
        className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      <span className="min-w-0">
        {children}
        {hint ? (
          <span className="mt-1 block font-prose text-meta text-mute">
            {hint}
          </span>
        ) : null}
      </span>
    </label>
  );
}
