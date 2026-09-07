"use client";

import { useEffect, useState } from "react";
import { readPostDestinations } from "@/lib/api/posts";
import type { PublicationDestinationChoice } from "@/lib/api/query";
import {
  eventWord,
  offeredFor,
  type Transition,
  transitionEvent,
} from "@/lib/publication-delivery";
import styles from "./AnnouncementChoice.module.css";

const QUIET: Record<Transition, string> = {
  publish:
    "Nothing is sent. The post still appears on the blog and in the feeds.",
  changes: "Nothing is sent. The changes still go live.",
  withdraw: "Nothing is sent. The post still comes down.",
  republish: "Nothing is sent. The post still goes back up.",
};

const SENT: Record<Transition, string> = {
  publish: "Each one receives a summary and a link, never the article itself.",
  changes:
    "Each one receives the same summary again, with the new edition's id.",
  withdraw: "Each one is told the post came down, and nothing about why.",
  republish:
    "Each one receives the summary again for the edition going back up.",
};

/** Where one public transition announces, and the line it may say alongside. */
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
      <p className={styles.quiet}>
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
    <div className={styles.where}>
      <fieldset className={styles.choice}>
        <legend>Where</legend>
        <ul>
          {takers.map((one) => (
            <li key={one.id}>
              <label className={styles.line}>
                <input
                  checked={picked.includes(one.id)}
                  onChange={(event) => toggle(one.id, event.target.checked)}
                  type="checkbox"
                />
                <span>{one.name}</span>
              </label>
              {one.role && picked.includes(one.id) ? (
                <label className={`${styles.line} ${styles.ping}`}>
                  <input
                    checked={pinging.includes(one.id)}
                    onChange={(event) =>
                      togglePing(one.id, event.target.checked)
                    }
                    type="checkbox"
                  />
                  <span>
                    Ping @{one.role}
                    <span>
                      Everyone with that role gets a notification. It cannot be
                      taken back.
                    </span>
                  </span>
                </label>
              ) : null}
            </li>
          ))}
        </ul>
        <p className={styles.quiet}>
          {picked.length === 0 ? QUIET[transition] : SENT[transition]}
        </p>
      </fieldset>

      {picked.length > 0 ? (
        <label className={styles.note}>
          <span>Note</span>
          <textarea
            maxLength={500}
            onChange={(event) => onNote(event.target.value)}
            placeholder="One line of context every one of them receives."
            rows={2}
            value={note}
          />
          <span className={styles.aside}>
            It travels with the announcement and never becomes part of the post.
          </span>
        </label>
      ) : null}
    </div>
  );
}
