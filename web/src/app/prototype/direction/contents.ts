"use client";

import { useEffect, useState } from "react";
import type { Body } from "./data";

export type Heading = { id: string; text: string; number: number };

export function headings(blocks: Body[] | undefined): Heading[] {
  if (!blocks) return [];
  let number = 0;
  return blocks.flatMap((block) => {
    if (block.kind !== "h") return [];
    number += 1;
    return [{ id: `section-${number}`, text: block.text, number }];
  });
}

/** Reports which recorded heading the reader is nearest, for the contents rail */
export function useReadingPosition(list: Heading[]) {
  const [active, setActive] = useState(list[0]?.id ?? "");

  useEffect(() => {
    if (!list.length) return;
    const nodes = list
      .map((entry) => document.getElementById(entry.id))
      .filter((node): node is HTMLElement => Boolean(node));
    if (!nodes.length) return;

    const observer = new IntersectionObserver(
      (records) => {
        const visible = records
          .filter((record) => record.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]?.target.id) setActive(visible[0].target.id);
      },
      { rootMargin: "-12% 0px -70% 0px", threshold: 0 },
    );
    for (const node of nodes) observer.observe(node);
    return () => observer.disconnect();
  }, [list]);

  return active;
}
