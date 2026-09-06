"use client";

import { useEffect, useState } from "react";
import { readPostDeliveries } from "@/lib/api/posts";
import type { PostDelivery } from "@/lib/api/query";
import { shortMoment } from "@/lib/dates";
import styles from "./DeliveryBand.module.css";

export function DeliveryBand({ postId }: { postId: string }) {
  const [sent, setSent] = useState<PostDelivery[]>([]);

  useEffect(() => {
    let live = true;
    void readPostDeliveries(postId).then((answer) => {
      if (live && answer.value) setSent(answer.value.deliveries);
    });
    return () => {
      live = false;
    };
  }, [postId]);

  if (sent.length === 0) return null;

  return (
    <section aria-labelledby="sent-to" className={styles.band}>
      <p className={styles.mark} id="sent-to">
        Sent
      </p>
      <ul className={styles.list}>
        {sent.map((one) => (
          <li className={styles.row} key={one.id} data-state={one.state}>
            <span className={styles.where}>{one.destination}</span>
            <span className={styles.what}>{transition(one.eventType)}</span>
            <span className={styles.standing}>{standing(one)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}

// transition names the public change this delivery is about, in the words the
// editor uses everywhere else.
function transition(type: string): string {
  if (type.endsWith("updated.v1")) return "Changes";
  if (type.endsWith("withdrawn.v1")) return "Withdrawal";
  return "Publication";
}

// standing says how one delivery ended, without a response body or an address.
function standing(one: PostDelivery): string {
  const when = shortMoment(one.settledAt ?? one.occurredAt);
  if (one.state === "delivered") return `Arrived ${when}`;
  if (one.state === "failed") {
    return `${one.last?.detail ?? "It did not arrive."} ${when}`;
  }
  return "On its way";
}
