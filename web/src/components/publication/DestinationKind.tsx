"use client";

import { Hash, Webhook } from "lucide-react";
import type { PublicationDestinationKind } from "@/lib/api/query";
import styles from "./DestinationKind.module.css";

const KINDS: {
  kind: PublicationDestinationKind;
  word: string;
  what: string;
}[] = [
  {
    kind: "discord",
    word: "Discord",
    what: "One channel. Illarin writes the announcement and sends it once, when a post first goes live.",
  },
  {
    kind: "webhook",
    word: "Webhook",
    what: "An endpoint of your own. It receives a signed summary of every transition it asks for.",
  },
];

/** What a new destination is, chosen before anything is asked about it. */
export function DestinationKind({
  chosen,
  onChosen,
}: {
  chosen: PublicationDestinationKind;
  onChosen: (kind: PublicationDestinationKind) => void;
}) {
  return (
    <fieldset className={styles.choice}>
      <legend>What it is</legend>
      <div className={styles.pair}>
        {KINDS.map((one) => (
          <label
            className={styles.tile}
            data-on={one.kind === chosen || undefined}
            key={one.kind}
          >
            <input
              checked={one.kind === chosen}
              name="destination-kind"
              onChange={() => onChosen(one.kind)}
              type="radio"
            />
            <span className={styles.mark}>
              {one.kind === "discord" ? (
                <Hash size={17} strokeWidth={2} aria-hidden="true" />
              ) : (
                <Webhook size={17} strokeWidth={1.8} aria-hidden="true" />
              )}
            </span>
            <span className={styles.word}>{one.word}</span>
            <span className={styles.what}>{one.what}</span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}
