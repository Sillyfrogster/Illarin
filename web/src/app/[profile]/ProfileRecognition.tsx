"use client";

import Image from "next/image";
import { useEffect, useRef, useState } from "react";
import type { ProfileDistinction } from "@/lib/api/query";
import styles from "./ProfileRecognition.module.css";

/** How many the band shows before it hands the rest to the dialog. */
const SHOWN = 4;

export function ProfileRecognition({
  titles,
  badges,
}: {
  titles: ProfileDistinction[];
  badges: ProfileDistinction[];
}) {
  const [open, setOpen] = useState(false);
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const element = dialog.current;
    if (!element) return;
    if (open && !element.open) element.showModal();
    if (!open && element.open) element.close();
  }, [open]);

  const all = [...badges, ...titles];
  if (all.length === 0) return null;
  const shown = all.slice(0, SHOWN);
  const hidden = all.length - shown.length;

  return (
    <>
      <ul className={styles.strip}>
        {shown.map((one) => (
          <li className={styles.chip} key={one.id}>
            <Mark one={one} size={20} />
            {one.name}
          </li>
        ))}
        {hidden > 0 || all.length > 0 ? (
          <li>
            <button
              type="button"
              className={styles.more}
              onClick={() => setOpen(true)}
            >
              {hidden > 0 ? `${hidden} more` : "What these are"}
            </button>
          </li>
        ) : null}
      </ul>

      <dialog
        className={styles.dialog}
        ref={dialog}
        onClose={() => setOpen(false)}
        aria-labelledby="recognition-heading"
      >
        <h2 id="recognition-heading">Given by Illarin</h2>
        {badges.length > 0 ? <Listing heading="Badges" all={badges} /> : null}
        {titles.length > 0 ? <Listing heading="Titles" all={titles} /> : null}
        <button
          type="button"
          className={styles.close}
          onClick={() => setOpen(false)}
        >
          Close
        </button>
      </dialog>
    </>
  );
}

function Listing({
  heading,
  all,
}: {
  heading: string;
  all: ProfileDistinction[];
}) {
  return (
    <section className={styles.listing}>
      <h3>{heading}</h3>
      <ul>
        {all.map((one) => (
          <li key={one.id}>
            <Mark one={one} size={32} />
            <span className={styles.entry}>
              <strong>{one.name}</strong>
              {one.explanation ? <span>{one.explanation}</span> : null}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function Mark({ one, size }: { one: ProfileDistinction; size: number }) {
  if (!one.mark) return null;
  return (
    <Image
      className={styles.mark}
      src={one.mark.url}
      alt=""
      width={size}
      height={size}
      style={{ width: size, height: size }}
      unoptimized
    />
  );
}
