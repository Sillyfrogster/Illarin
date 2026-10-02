"use client";

import {
  type PointerEvent as ReactPointerEvent,
  type RefObject,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { WorkBlock } from "@/lib/api/query";
import {
  BLOCK_GRID_GAP_PX,
  type BlockLayout,
  type BlockWidth,
  WIDTH_LABELS,
} from "@/lib/page-arrangement";
import { moveBlock } from "./composition";
import { dropPosition, snapWidth } from "./gestures";
import { useWorkspace } from "./state";

/** How far the pointer travels before a press on a block becomes a lift, so a click still clicks. */
const LIFT_AFTER_PX = 5;
/** How close to the top or bottom of the screen a held block starts the page scrolling. */
const EDGE_PX = 72;
/** How far above a block its drag handle reaches, so holding a block by its handle counts as being in its row. */
const HANDLE_REACH_PX = 48;
/** How far the pointer must travel after a reorder before the next one, so a block on the line does not flicker between two places. */
const SETTLE_PX = 10;

export type Lift = { blockId: string; title: string; width: BlockWidth };

export type Stretch = { blockId: string; width: BlockWidth };

export type Arranging = {
  shown: WorkBlock[];
  lift: Lift | null;
  liftCard: RefObject<HTMLDivElement | null>;
  stretch: Stretch | null;
  pressBlock: (
    event: ReactPointerEvent,
    block: WorkBlock,
    onClick?: () => void,
  ) => void;
  pressEdge: (
    event: ReactPointerEvent,
    block: WorkBlock,
    startColumn: number,
  ) => void;
};

function blockNodes(): HTMLElement[] {
  return [...document.querySelectorAll<HTMLElement>("[data-arrange-id]")];
}

/** useArranging moves blocks by dragging them and resizes them by dragging their right edge, showing the page as it will be while the pointer is down. */
export function useArranging(): Arranging {
  const workspace = useWorkspace();
  const { arrangement, blocks } = workspace;
  const [order, setOrder] = useState<string[] | null>(null);
  const [lift, setLift] = useState<Lift | null>(null);
  const [stretch, setStretch] = useState<Stretch | null>(null);
  const liftCard = useRef<HTMLDivElement>(null);
  const settledAt = useRef<{ x: number; y: number } | null>(null);
  const anchor = useRef<{
    blockId: string;
    pointer: { x: number; y: number };
    top: number;
  } | null>(null);

  // Blocks that fold while one is held must not pull the held one away from the pointer
  // biome-ignore lint/correctness/useExhaustiveDependencies: The held block is measured again each time it is lifted or dropped.
  useLayoutEffect(() => {
    const held = anchor.current;
    if (!held) return;
    anchor.current = null;
    const top = document
      .getElementById(`block-${held.blockId}`)
      ?.getBoundingClientRect().top;
    const shift = top === undefined ? 0 : top - held.top;
    window.scrollBy({ behavior: "instant", top: shift });
    settledAt.current = { x: held.pointer.x, y: held.pointer.y + shift };
  }, [lift]);
  const wasBusy = useRef(false);

  // A dropped order stays on screen until the server's answer replaces it
  useEffect(() => {
    if (!order || lift) return;
    const same =
      blocks.length === order.length &&
      blocks.every((block, index) => block.id === order[index]);
    if (same || (wasBusy.current && !arrangement.busy)) setOrder(null);
    wasBusy.current = arrangement.busy;
  }, [arrangement.busy, blocks, lift, order]);

  const shown = useMemo(() => {
    const byId = new Map(blocks.map((block) => [block.id, block]));
    const ordered = order
      ? order.flatMap((id) => {
          const block = byId.get(id);
          return block ? [block] : [];
        })
      : blocks;
    return stretch
      ? ordered.map((block) =>
          block.id === stretch.blockId
            ? { ...block, width: stretch.width }
            : block,
        )
      : ordered;
  }, [blocks, order, stretch]);

  function pressBlock(
    event: ReactPointerEvent,
    block: WorkBlock,
    onClick?: () => void,
  ) {
    if (event.button !== 0 || arrangement.busy || blocks.length < 2) {
      if (event.button === 0 && onClick) onClick();
      return;
    }
    const start = { x: event.clientX, y: event.clientY };
    const from = blocks.findIndex((one) => one.id === block.id);
    let lifted = false;
    let to = from;
    let frame = 0;
    let edge = 0;
    let pointer = start;

    const place = (x: number, y: number) => {
      const card = liftCard.current;
      if (card) card.style.translate = `${x}px ${y}px`;
      else
        window.requestAnimationFrame(() => {
          if (liftCard.current)
            liftCard.current.style.translate = `${x}px ${y}px`;
        });
    };
    const follow = (moved: PointerEvent) => {
      if (!lifted) {
        if (
          Math.hypot(moved.clientX - start.x, moved.clientY - start.y) <
          LIFT_AFTER_PX
        )
          return;
        lifted = true;
        settledAt.current = {
          x: moved.clientX,
          y: moved.clientY + window.scrollY,
        };
        anchor.current = {
          blockId: block.id,
          pointer: settledAt.current,
          top:
            document
              .getElementById(`block-${block.id}`)
              ?.getBoundingClientRect().top ?? 0,
        };
        document.body.dataset.lifting = "true";
        setOrder(blocks.map((one) => one.id));
        setLift({ blockId: block.id, title: block.title, width: block.width });
      }
      moved.preventDefault();
      pointer = { x: moved.clientX, y: moved.clientY };
      place(moved.clientX - 28, moved.clientY - 22);
      if (frame) return;
      frame = window.requestAnimationFrame(() => {
        frame = 0;
        reconsider();
      });
      if (!edge) edge = window.requestAnimationFrame(scrollAtEdge);
    };

    // Measured on the page, not the screen, so scrolling under a still pointer counts as moving
    const reconsider = () => {
      const here = { x: pointer.x, y: pointer.y + window.scrollY };
      const last = settledAt.current;
      if (last && Math.hypot(here.x - last.x, here.y - last.y) < SETTLE_PX)
        return;
      const rest = blockNodes()
        .filter((one) => one.dataset.arrangeId !== block.id)
        .map((one) => one.getBoundingClientRect());
      const next = dropPosition(rest, pointer, HANDLE_REACH_PX);
      if (next === to) return;
      to = next;
      settledAt.current = here;
      setOrder(moveBlock(blocks, block.id, next).map((one) => one.id));
    };

    const scrollAtEdge = () => {
      edge = 0;
      if (!lifted) return;
      const top = pointer.y - EDGE_PX * 1.5;
      const bottom = pointer.y - (window.innerHeight - EDGE_PX);
      const speed =
        top < 0
          ? Math.max(top / 4, -24)
          : bottom > 0
            ? Math.min(bottom / 4, 24)
            : 0;
      if (speed) {
        window.scrollBy({ behavior: "instant", top: speed });
        reconsider();
      }
      edge = window.requestAnimationFrame(scrollAtEdge);
    };

    const finish = (cancelled: boolean) => {
      window.removeEventListener("pointermove", follow);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("pointercancel", cancel);
      window.removeEventListener("keydown", onEscape, true);
      if (frame) window.cancelAnimationFrame(frame);
      if (edge) window.cancelAnimationFrame(edge);
      delete document.body.dataset.lifting;
      if (!lifted) {
        if (!cancelled) onClick?.();
        return;
      }
      anchor.current = {
        blockId: block.id,
        pointer: settledAt.current ?? start,

        top:
          document.getElementById(`block-${block.id}`)?.getBoundingClientRect()
            .top ?? 0,
      };
      setLift(null);
      if (cancelled || to === from) {
        setOrder(null);
        return;
      }
      arrangement.move(block.id, to);
      workspace.say(`“${block.title}” moved to ${to + 1} of ${blocks.length}.`);
    };
    const up = () => finish(false);
    const cancel = () => finish(true);
    const onEscape = (key: KeyboardEvent) => {
      if (key.key !== "Escape") return;
      key.preventDefault();
      key.stopPropagation();
      finish(true);
    };

    window.addEventListener("pointermove", follow, { passive: false });
    window.addEventListener("pointerup", up);
    window.addEventListener("pointercancel", cancel);
    window.addEventListener("keydown", onEscape, true);
  }

  function pressEdge(
    event: ReactPointerEvent,
    block: WorkBlock,
    startColumn: number,
  ) {
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    const grid = document
      .querySelector<HTMLElement>("[data-block-grid]")
      ?.getBoundingClientRect();
    if (!grid) return;
    const measure = (x: number) =>
      snapWidth({
        edge: x,
        gap: BLOCK_GRID_GAP_PX * (grid.width / gridLayoutWidth()),
        layout: block.layout as BlockLayout,
        left: grid.left,
        startColumn,
        width: grid.width,
      });
    let width = block.width;
    document.body.dataset.stretching = "true";
    setStretch({ blockId: block.id, width });

    const follow = (moved: PointerEvent) => {
      moved.preventDefault();
      const next = measure(moved.clientX);
      if (next === width) return;
      width = next;
      setStretch({ blockId: block.id, width });
    };
    const finish = (cancelled: boolean) => {
      window.removeEventListener("pointermove", follow);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("pointercancel", cancel);
      window.removeEventListener("keydown", onEscape, true);
      delete document.body.dataset.stretching;
      setStretch(null);
      if (cancelled || width === block.width) return;
      workspace.writeBlock({ ...block, width });
      workspace.say(
        `“${block.title}” is now ${WIDTH_LABELS[width].toLowerCase()} width.`,
      );
    };
    const up = () => finish(false);
    const cancel = () => finish(true);
    const onEscape = (key: KeyboardEvent) => {
      if (key.key !== "Escape") return;
      key.preventDefault();
      key.stopPropagation();
      finish(true);
    };
    window.addEventListener("pointermove", follow, { passive: false });
    window.addEventListener("pointerup", up);
    window.addEventListener("pointercancel", cancel);
    window.addEventListener("keydown", onEscape, true);
  }

  return { lift, liftCard, pressBlock, pressEdge, shown, stretch };
}

/** gridLayoutWidth is the block grid's own width before any zoom, so its gaps can be scaled to what is on screen. */
function gridLayoutWidth(): number {
  const grid = document.querySelector<HTMLElement>("[data-block-grid]");
  return grid?.offsetWidth || 1;
}
