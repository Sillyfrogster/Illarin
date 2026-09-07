"use client";

import type { PublicationEvent } from "@/lib/api/query";
import { EVENT_WORDS, EVENTS } from "@/lib/publication-delivery";
import styles from "./EventChoice.module.css";

/** Which public transitions one endpoint receives. */
export function EventChoice({
  chosen,
  onChosen,
}: {
  chosen: PublicationEvent[];
  onChosen: (chosen: PublicationEvent[]) => void;
}) {
  function toggle(event: PublicationEvent, on: boolean) {
    onChosen(
      EVENTS.filter((one) => (one === event ? on : chosen.includes(one))),
    );
  }

  return (
    <fieldset className={styles.choice}>
      <legend>What it receives</legend>
      <ul>
        {EVENTS.map((one) => (
          <li key={one}>
            <label className={styles.line}>
              <input
                checked={chosen.includes(one)}
                onChange={(event) => toggle(one, event.target.checked)}
                type="checkbox"
              />
              <span>
                {EVENT_WORDS[one].word}
                <span>{EVENT_WORDS[one].what}</span>
              </span>
            </label>
          </li>
        ))}
      </ul>
      {chosen.length === 0 ? (
        <p className={styles.none}>
          It receives nothing. Switch the destination off instead if that is
          what you mean.
        </p>
      ) : null}
    </fieldset>
  );
}
