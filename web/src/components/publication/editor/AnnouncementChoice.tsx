"use client";

import { useEffect, useState } from "react";
import { readPostDestinations } from "@/lib/api/posts";
import type { PublicationDestinationChoice } from "@/lib/api/query";
import styles from "./AnnouncementChoice.module.css";

/** Where a publication announces, and the one line it may say alongside it. */
export function AnnouncementChoice({
  postId,
  chosen,
  note,
  onChosen,
  onNote,
}: {
  postId: string;
  chosen: string[] | null;
  note: string;
  onChosen: (chosen: string[]) => void;
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

  if (offered.length === 0) return null;

  const picked =
    chosen ?? offered.filter((one) => one.byDefault).map((one) => one.id);

  function toggle(id: string, on: boolean) {
    onChosen(on ? [...picked, id] : picked.filter((held) => held !== id));
  }

  return (
    <div className={styles.where}>
      <fieldset className={styles.choice}>
        <legend>Where</legend>
        <ul>
          {offered.map((one) => (
            <li key={one.id}>
              <label className={styles.line}>
                <input
                  checked={picked.includes(one.id)}
                  onChange={(event) => toggle(one.id, event.target.checked)}
                  type="checkbox"
                />
                <span>{one.name}</span>
              </label>
            </li>
          ))}
        </ul>
        <p className={styles.quiet}>
          {picked.length === 0
            ? "Nothing is sent. The post still appears on the blog and in the feeds."
            : "Each one receives a summary and a link, never the article itself."}
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
