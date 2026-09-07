"use client";

import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  CircleSlash,
  Clock,
} from "lucide-react";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Section } from "@/components/console/Section";
import { readDeliveryAttempts, replayDelivery } from "@/lib/api/publication";
import type {
  PostDelivery,
  PostDeliveryAttempt,
  PostDeliveryState,
} from "@/lib/api/query";
import { shortMoment } from "@/lib/dates";
import {
  type DeliveryState,
  deliveryStanding,
  deliveryState,
  eventWord,
} from "@/lib/publication-delivery";
import styles from "./DeliveryList.module.css";

/** The views the authority narrows delivery work to. */
const VIEWS: { key: string; word: string; state?: PostDeliveryState }[] = [
  { key: "all", word: "All" },
  { key: "failed", word: "Stopped", state: "failed" },
  { key: "pending", word: "Waiting", state: "pending" },
  { key: "delivered", word: "Arrived", state: "delivered" },
  { key: "unconfirmed", word: "Unconfirmed", state: "unconfirmed" },
];

export function DeliveryList({
  deliveries,
  view,
  onView,
  onChanged,
  onFailure,
}: {
  deliveries: PostDelivery[];
  view: string;
  onView: (view: string, state?: PostDeliveryState) => void;
  onChanged: (delivery: PostDelivery) => void;
  onFailure: (message: string) => void;
}) {
  const [tried, setTried] = useState<Record<string, PostDeliveryAttempt[]>>({});
  const [working, setWorking] = useState("");

  async function open(one: PostDelivery) {
    if (tried[one.id]) return;
    const answer = await readDeliveryAttempts(one.id);
    const made = answer.value?.attempts;
    if (made) setTried((held) => ({ ...held, [one.id]: made }));
  }

  async function again(one: PostDelivery) {
    setWorking(one.id);
    const answer = await replayDelivery(one.id);
    setWorking("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    setTried((held) => ({ ...held, [one.id]: [] }));
    onChanged(answer.value);
  }

  return (
    <Section id="deliveries" title="Deliveries" count={deliveries.length}>
      <fieldset className={styles.views}>
        <legend>Narrow deliveries</legend>
        {VIEWS.map((one) => (
          <button
            aria-pressed={view === one.key}
            className={styles.view}
            key={one.key}
            onClick={() => onView(one.key, one.state)}
            type="button"
          >
            {one.word}
          </button>
        ))}
      </fieldset>

      {deliveries.length === 0 ? (
        <p className={rows.empty}>{nothingHere(view)}</p>
      ) : (
        <ul className={rows.list}>
          {deliveries.map((one) => {
            const state = deliveryState(one);
            const attempts = tried[one.id];
            return (
              <li className={styles.row} data-state={state} key={one.id}>
                <span className={styles.state}>
                  <StateMark state={state} />
                </span>
                <span className={rows.name}>
                  {one.postTitle}
                  <span className={styles.going}>
                    {one.destination} · {eventWord(one.eventType)}
                  </span>
                </span>
                <span className={rows.detail}>{deliveryStanding(one)}</span>
                <span className={styles.actions}>
                  {state === "gaveUp" && !one.removed ? (
                    <button
                      className={rows.textButton}
                      disabled={working === one.id}
                      onClick={() => void again(one)}
                      type="button"
                    >
                      {working === one.id ? "Sending…" : "Send again"}
                    </button>
                  ) : null}
                </span>
                {one.attempts > 0 ? (
                  <details
                    className={styles.history}
                    onToggle={() => void open(one)}
                  >
                    <summary>
                      {one.attempts === 1
                        ? "1 attempt"
                        : `${one.attempts} attempts`}
                      {one.run > 1 ? ` over ${one.run} runs` : ""}
                    </summary>
                    {attempts ? (
                      <ol className={styles.attempts}>
                        {attempts.map((made) => (
                          <li key={made.number}>
                            <span className={styles.number}>
                              {made.run > 1 ? `${made.run}.` : ""}
                              {made.number}
                            </span>
                            <span
                              className={styles.outcome}
                              data-outcome={made.outcome}
                            >
                              {made.detail || "It took it."}
                            </span>
                            <span className={styles.when}>
                              {shortMoment(made.attemptedAt)} · {made.tookMs}ms
                            </span>
                          </li>
                        ))}
                      </ol>
                    ) : (
                      <p className={styles.reading}>Reading the attempts…</p>
                    )}
                  </details>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </Section>
  );
}

function StateMark({ state }: { state: DeliveryState }) {
  if (state === "arrived") {
    return <CircleCheck size={18} strokeWidth={1.8} aria-hidden="true" />;
  }
  if (state === "gaveUp") {
    return <CircleAlert size={18} strokeWidth={1.8} aria-hidden="true" />;
  }
  if (state === "stopped") {
    return <CircleSlash size={18} strokeWidth={1.8} aria-hidden="true" />;
  }
  if (state === "unconfirmed") {
    return <CircleHelp size={18} strokeWidth={1.8} aria-hidden="true" />;
  }
  return <Clock size={18} strokeWidth={1.8} aria-hidden="true" />;
}

function nothingHere(view: string): string {
  if (view === "failed") return "Nothing has stopped short.";
  if (view === "pending") return "Nothing is on its way.";
  if (view === "delivered") return "Nothing has arrived yet.";
  if (view === "unconfirmed") {
    return "Every announcement Discord took, it confirmed.";
  }
  return "No post has announced anywhere yet.";
}
