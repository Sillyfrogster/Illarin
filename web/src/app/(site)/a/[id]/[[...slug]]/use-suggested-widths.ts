"use client";

import { type RefObject, useCallback, useEffect, useState } from "react";
import type { AssetBlock } from "@/lib/api/query";
import {
  type BlockWidth,
  suggestedBlockWidth,
  suggestionCandidateWidths,
} from "@/lib/page-arrangement";

/** Renders a block at each width it could take, to find the one that wastes least. */
function measureCandidateHeights(
  source: HTMLElement,
  layout: AssetBlock["layout"],
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

/** The width each block would rather have, remeasured whenever the page settles. */
export function useSuggestedWidths({
  availableWidth,
  blocks,
  paused,
  rows,
}: {
  availableWidth: number | undefined;
  blocks: AssetBlock[];
  paused: boolean;
  rows: RefObject<HTMLDivElement | null>;
}): Record<string, BlockWidth> {
  const [suggested, setSuggested] = useState<Record<string, BlockWidth>>({});

  const measure = useCallback(() => {
    if (!rows.current || availableWidth === undefined) return;
    const next: Record<string, BlockWidth> = {};
    const byId = new Map(blocks.map((block) => [block.id, block]));
    for (const node of rows.current.querySelectorAll<HTMLElement>(
      "[data-block-id]",
    )) {
      const blockId = node.dataset.blockId;
      const content = node.querySelector<HTMLElement>("[data-block-content]");
      const block = blockId ? byId.get(blockId) : undefined;
      if (!blockId || !block || block.isEmpty || !content) continue;
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
      if (suggestion) next[blockId] = suggestion;
    }
    setSuggested((current) =>
      sameSuggestions(current, next) ? current : next,
    );
  }, [availableWidth, blocks, rows]);

  useEffect(() => {
    if (paused || !rows.current) return;
    let frame = window.requestAnimationFrame(measure);
    const observer = new ResizeObserver(() => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(measure);
    });
    observer.observe(rows.current);
    for (const content of rows.current.querySelectorAll<HTMLElement>(
      "[data-block-content]",
    )) {
      observer.observe(content);
    }
    return () => {
      window.cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, [measure, paused, rows]);

  return suggested;
}
