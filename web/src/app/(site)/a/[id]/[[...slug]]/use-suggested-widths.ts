"use client";

import { type RefObject, useCallback, useEffect, useState } from "react";
import type { WorkBlock } from "@/lib/api/query";
import {
  type BlockWidth,
  suggestedBlockWidth,
  suggestionCandidateWidths,
} from "@/lib/page-arrangement";

function measureCandidateHeights(
  source: HTMLElement,
  layout: WorkBlock["layout"],
  availableWidth: number,
): Partial<Record<BlockWidth, number>> {
  const heights: Partial<Record<BlockWidth, number>> = {};

  for (const candidate of suggestionCandidateWidths(layout, availableWidth)) {
    const clone = source.cloneNode(true) as HTMLElement;
    clone.removeAttribute("id");
    clone.setAttribute("aria-hidden", "true");
    clone.inert = true;
    Object.assign(clone.style, {
      gridColumn: "auto",
      inset: "0 auto auto -100000px",
      maxWidth: "none",
      pointerEvents: "none",
      position: "fixed",
      visibility: "hidden",
      width: `${candidate.renderedWidth}px`,
    });
    for (const identified of clone.querySelectorAll("[id]")) {
      identified.removeAttribute("id");
    }
    for (const ignored of clone.querySelectorAll(
      "[data-empty], [data-measurement-ignore]",
    )) {
      ignored.remove();
    }

    document.body.append(clone);
    for (const excerpt of clone.querySelectorAll<HTMLElement>(
      "[data-line-excerpt]",
    )) {
      const isCut = excerpt.scrollHeight - excerpt.clientHeight > 1;
      const sibling = excerpt.nextElementSibling;
      const readMore =
        sibling instanceof HTMLElement && sibling.hasAttribute("data-read-more")
          ? sibling
          : null;
      if (!isCut) {
        readMore?.remove();
      } else if (!readMore) {
        const readMoreSpace = document.createElement("span");
        readMoreSpace.style.display = "block";
        readMoreSpace.style.height = "20px";
        excerpt.after(readMoreSpace);
      }
    }

    const content = clone.querySelector<HTMLElement>("[data-block-content]");
    const height = content?.getBoundingClientRect().height ?? 0;
    if (height > 0) heights[candidate.width] = height;
    clone.remove();
  }

  return heights;
}

function sameSuggestions(
  current: Record<string, BlockWidth>,
  next: Record<string, BlockWidth>,
) {
  const currentIds = Object.keys(current);
  const nextIds = Object.keys(next);
  return (
    currentIds.length === nextIds.length &&
    currentIds.every((id) => current[id] === next[id])
  );
}

/** How long the page must stop changing size before the widths are measured again. */
const REMEASURE_AFTER_MS = 500;
/** The least idle time worth starting another block's measurement in. */
const IDLE_FLOOR_MS = 4;

function whenIdle(step: (deadline: IdleDeadline) => void): number {
  if (typeof window.requestIdleCallback === "function")
    return window.requestIdleCallback(step, { timeout: 2000 });
  return window.setTimeout(
    () => step({ didTimeout: false, timeRemaining: () => 8 }),
    50,
  );
}

function cancelIdle(task: number) {
  if (typeof window.cancelIdleCallback === "function")
    window.cancelIdleCallback(task);
  clearTimeout(task);
}

export function useSuggestedWidths({
  availableWidth,
  blocks,
  paused,
  rows,
}: {
  availableWidth: number | undefined;
  blocks: WorkBlock[];
  paused: boolean;
  rows: RefObject<HTMLDivElement | null>;
}): Record<string, BlockWidth> {
  const [suggested, setSuggested] = useState<Record<string, BlockWidth>>({});

  // Measured a few blocks at a time while the page is idle, so it never holds up typing or a transition
  const measure = useCallback(() => {
    if (!rows.current || availableWidth === undefined) return () => {};
    const next: Record<string, BlockWidth> = {};
    const byId = new Map(blocks.map((block) => [block.id, block]));
    const nodes = [
      ...rows.current.querySelectorAll<HTMLElement>("[data-block-id]"),
    ];
    let at = 0;
    let task = 0;
    const step = (deadline: IdleDeadline) => {
      while (at < nodes.length && deadline.timeRemaining() > IDLE_FLOOR_MS) {
        const node = nodes[at++];
        const block = byId.get(node.dataset.blockId ?? "");
        if (!block || block.isEmpty) continue;
        if (!node.querySelector("[data-block-content]")) continue;
        const suggestion = suggestedBlockWidth({
          availableWidth,
          layout: block.layout,
          renderedHeights: measureCandidateHeights(
            node,
            block.layout,
            availableWidth,
          ),
          width: block.width,
        });
        if (suggestion) next[block.id] = suggestion;
      }
      if (at < nodes.length) {
        task = whenIdle(step);
        return;
      }
      setSuggested((current) =>
        sameSuggestions(current, next) ? current : next,
      );
    };
    task = whenIdle(step);
    return () => cancelIdle(task);
  }, [availableWidth, blocks, rows]);

  useEffect(() => {
    if (paused || !rows.current) return;
    let stop = measure();
    let wait = 0;
    const observer = new ResizeObserver(() => {
      stop();
      window.clearTimeout(wait);
      wait = window.setTimeout(() => {
        stop = measure();
      }, REMEASURE_AFTER_MS);
    });
    observer.observe(rows.current);
    for (const content of rows.current.querySelectorAll<HTMLElement>(
      "[data-block-content]",
    )) {
      observer.observe(content);
    }
    return () => {
      stop();
      window.clearTimeout(wait);
      observer.disconnect();
    };
  }, [measure, paused, rows]);

  return suggested;
}
