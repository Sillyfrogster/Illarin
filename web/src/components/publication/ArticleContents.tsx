"use client";

import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";
import type { PostContentsEntry } from "@/lib/post-contents";
import styles from "./ArticleAside.module.css";

/** How far down the window a heading has to pass before the contents call its section the one being read. */
const READING_LINE = 0.18;

/** The outline a post's own headings make, in the margin on a wide screen and a disclosure on a narrow one. */
export function ArticleContents({ entries }: { entries: PostContentsEntry[] }) {
  const [open, setOpen] = useState(false);
  const [here, setHere] = useState("");

  useEffect(() => {
    const marks = entries
      .map((entry) => document.getElementById(entry.anchor))
      .filter((mark): mark is HTMLElement => mark !== null);
    if (marks.length === 0) return;
    let asked = 0;
    const settle = () => {
      asked = 0;
      const line = window.innerHeight * READING_LINE;
      let reached = "";
      for (const mark of marks) {
        if (mark.getBoundingClientRect().top <= line) reached = mark.id;
      }
      setHere(reached);
    };
    const ask = () => {
      if (asked === 0) asked = requestAnimationFrame(settle);
    };
    settle();
    window.addEventListener("scroll", ask, { passive: true });
    window.addEventListener("resize", ask);
    return () => {
      if (asked !== 0) cancelAnimationFrame(asked);
      window.removeEventListener("scroll", ask);
      window.removeEventListener("resize", ask);
    };
  }, [entries]);

  return (
    <nav aria-label="Contents" className={styles.contents} data-open={open}>
      <p aria-hidden="true" className={styles.label}>
        Contents
      </p>
      <button
        aria-controls="article-contents"
        aria-expanded={open}
        className={styles.disclosure}
        onClick={() => setOpen(!open)}
        type="button"
      >
        Contents
        <ChevronDown
          aria-hidden="true"
          className={styles.chevron}
          size={16}
          strokeWidth={1.8}
        />
      </button>
      <ol className={styles.list} id="article-contents">
        {entries.map((entry) => (
          <li
            className={styles.entry}
            data-level={entry.level}
            key={entry.anchor}
          >
            <a
              aria-current={entry.anchor === here ? "location" : undefined}
              className={styles.reach}
              href={`#${entry.anchor}`}
              onClick={() => setOpen(false)}
            >
              {entry.label}
            </a>
          </li>
        ))}
      </ol>
    </nav>
  );
}
