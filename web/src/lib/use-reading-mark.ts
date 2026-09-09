"use client";

import { useEffect, useState } from "react";

// How far down the window a mark has to pass before a rail calls its section the one being read.
const READING_LINE = 0.18;

/** The last of these elements to have passed the reading line, so a rail can say where a reader is. */
export function useReadingMark(anchors: string[]): string {
  const [here, setHere] = useState("");
  // The anchors are rebuilt on every render, so the effect watches their text rather than the array.
  const identity = anchors.join(" ");

  useEffect(() => {
    const marks = identity
      .split(" ")
      .filter((anchor) => anchor.length > 0)
      .map((anchor) => document.getElementById(anchor))
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
  }, [identity]);

  return here;
}
