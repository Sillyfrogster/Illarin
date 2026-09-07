"use client";

import { useEffect, useState } from "react";
import { readPostDeliveries } from "@/lib/api/posts";
import type { PostDelivery } from "@/lib/api/query";
import {
  deliveryStanding,
  deliveryState,
  eventWord,
} from "@/lib/publication-delivery";
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
          <li
            className={styles.row}
            key={one.id}
            data-state={deliveryState(one)}
          >
            <span className={styles.where}>{one.destination}</span>
            <span className={styles.what}>{eventWord(one.eventType)}</span>
            <span className={styles.standing}>{deliveryStanding(one)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}
