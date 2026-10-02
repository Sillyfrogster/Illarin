"use client";

import {
  type PointerEvent as ReactPointerEvent,
  useCallback,
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

/** How far a mouse travels before a press on a block becomes a lift, so a click still clicks. */
const LIFT_AFTER_PX = 4;
/** How far a press may wander and still count as a click. */
const TAP_PX = 10;
/** How far the pointer must travel after a reorder before the next one, so a block on the line does not flicker between two places. */
const SETTLE_PX = 12;
/** How close to the top or bottom of the screen a held block starts the page scrolling. */
const EDGE_PX = 80;
/** How long a dropped block takes to settle into its place. */
const SETTLE_MS = 200;
/** How long the other blocks take to make room. */
const REFLOW_MS = 260;
const EASE = "cubic-bezier(0.22, 1, 0.36, 1)";

export type Stretch = { blockId: string; width: BlockWidth };

export type ArrangeHandlers = {
  pressBlock: (
    event: ReactPointerEvent,
    block: WorkBlock,
    onClick?: () => void,
  ) => void;
  pressGrip: (event: ReactPointerEvent, block: WorkBlock) => void;
  pressEdge: (
    event: ReactPointerEvent,
    block: WorkBlock,
    startColumn: number,
  ) => void;
};

export type Arranging = ArrangeHandlers & {
  shown: WorkBlock[];
  lifted: string | null;
  stretch: Stretch | null;
};

function blockNode(blockId: string): HTMLElement | null {
  return document.getElementById(`block-${blockId}`);
}

export function zoomOf(node: Element): number {
  return (node as { currentCSSZoom?: number }).currentCSSZoom ?? 1;
}

/** liftClone copies a block into a card that rides the pointer, drawn at the same size as the block on screen. */
function liftClone(node: HTMLElement): HTMLElement {
  const box = node.getBoundingClientRect();
  const zoom = zoomOf(node);
  const clone = node.cloneNode(true) as HTMLElement;
  clone.removeAttribute("id");
  clone.removeAttribute("data-arrange-id");
  clone.setAttribute("aria-hidden", "true");
  clone.inert = true;
  Object.assign(clone.style, {
    boxShadow: "0 24px 60px -16px rgb(0 0 0 / 0.55)",
    left: `${box.left / zoom}px`,
    margin: "0",
    pointerEvents: "none",
    position: "fixed",
    rotate: "-1.2deg",
    scale: "1.025",
    top: `${box.top / zoom}px`,
    viewTransitionName: "none",
    width: `${node.offsetWidth}px`,
    zIndex: "95",
    zoom: String(zoom),
  });
  clone.style.setProperty("--map-zoom", String(zoom));
  document.body.append(clone);
  return clone;
}

/** settleClone glides the carried card into the block's place, then removes it. */
function settleClone(clone: HTMLElement, blockId: string, done: () => void) {
  window.requestAnimationFrame(() => {
    const slot = blockNode(blockId)?.getBoundingClientRect();
    const zoom = Number(clone.style.zoom) || 1;
    const box = clone.getBoundingClientRect();
    const [x = 0, y = 0] = (clone.style.translate || "0px 0px")
      .split(" ")
      .map((part) => Number.parseFloat(part) || 0);
    const target = slot
      ? `${x + (slot.left - box.left) / zoom}px ${y + (slot.top - box.top) / zoom}px`
      : `${x}px ${y}px`;
    clone
      .animate(
        [
          { rotate: "-1.2deg", scale: "1.025", translate: `${x}px ${y}px` },
          { rotate: "0deg", scale: "1", translate: target },
        ],
        { duration: SETTLE_MS, easing: EASE, fill: "forwards" },
      )
      .finished.catch(() => undefined)
      .finally(() => {
        clone.remove();
        done();
      });
  });
}

function listen(
  follow: (event: PointerEvent) => void,
  finish: (cancelled: boolean) => void,
) {
  const onUp = () => finish(false);
  const onCancel = () => finish(true);
  const onEscape = (key: KeyboardEvent) => {
    if (key.key !== "Escape") return;
    key.preventDefault();
    key.stopPropagation();
    finish(true);
  };
  window.addEventListener("pointermove", follow, { passive: false });
  window.addEventListener("pointerup", onUp);
  window.addEventListener("pointercancel", onCancel);
  window.addEventListener("keydown", onEscape, true);
  return () => {
    window.removeEventListener("pointermove", follow);
    window.removeEventListener("pointerup", onUp);
    window.removeEventListener("pointercancel", onCancel);
    window.removeEventListener("keydown", onEscape, true);
  };
}

/** useArranging moves blocks by dragging them and resizes them by dragging their right edge, showing the page as it will be while the pointer is down. */
export function useArranging(): Arranging {
  const workspace = useWorkspace();
  const { arrangement, blocks } = workspace;
  const [order, setOrder] = useState<string[] | null>(null);
  const [lifted, setLifted] = useState<string | null>(null);
  const [stretch, setStretch] = useState<Stretch | null>(null);
  const latest = useRef(workspace);
  useLayoutEffect(() => {
    latest.current = workspace;
  });
  const wasBusy = useRef(false);

  // A dropped order stays on screen until the server's answer replaces it
  useEffect(() => {
    if (!order || lifted) return;
    const same =
      blocks.length === order.length &&
      blocks.every((block, index) => block.id === order[index]);
    if (same || (wasBusy.current && !arrangement.busy)) setOrder(null);
    wasBusy.current = arrangement.busy;
  }, [arrangement.busy, blocks, lifted, order]);

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

  // A finger on a block scrolls the page and a tap opens it; only the grip lifts at once
  const press = useCallback(
    (
      event: ReactPointerEvent,
      block: WorkBlock,
      onClick: (() => void) | undefined,
      grip: boolean,
    ) => {
      const page = latest.current;
      if (event.button !== 0) return;
      const touch = event.pointerType !== "mouse";
      if (grip) event.preventDefault();
      const fixed = page.arrangement.busy || page.blocks.length < 2;
      if (fixed && grip) return;
      const blocks = page.blocks;
      const start = { x: event.clientX, y: event.clientY };
      const from = blocks.findIndex((one) => one.id === block.id);
      let clone: HTMLElement | null = null;
      let to = from;
      let pointer = start;
      let settledAt = start;
      let frame = 0;
      let edge = 0;

      const place = () => {
        if (!clone) return;
        const zoom = Number(clone.style.zoom) || 1;
        clone.style.translate = `${(pointer.x - start.x) / zoom}px ${(pointer.y - start.y) / zoom}px`;
      };

      // Measured on the page, not the screen, so scrolling under a still pointer counts as moving
      const reconsider = () => {
        const here = { x: pointer.x, y: pointer.y + window.scrollY };
        if (Math.hypot(here.x - settledAt.x, here.y - settledAt.y) < SETTLE_PX)
          return;
        const rest = [
          ...document.querySelectorAll<HTMLElement>("[data-arrange-id]"),
        ]
          .filter((one) => one.dataset.arrangeId !== block.id)
          .map((one) => one.getBoundingClientRect());
        const next = dropPosition(rest, pointer);
        if (next === to) return;
        to = next;
        settledAt = here;
        setOrder(moveBlock(blocks, block.id, next).map((one) => one.id));
      };

      const scrollAtEdge = () => {
        const above = pointer.y - EDGE_PX * 1.5;
        const below = pointer.y - (window.innerHeight - EDGE_PX);
        const speed =
          above < 0
            ? Math.max(above / 5, -22)
            : below > 0
              ? Math.min(below / 5, 22)
              : 0;
        if (speed) {
          window.scrollBy({ behavior: "instant", top: speed });
          reconsider();
        }
        edge = window.requestAnimationFrame(scrollAtEdge);
      };

      const lift = () => {
        const node = blockNode(block.id);
        if (!node) return;
        clone = liftClone(node);
        settledAt = { x: pointer.x, y: pointer.y + window.scrollY };
        document.body.dataset.lifting = "true";
        setOrder(blocks.map((one) => one.id));
        setLifted(block.id);
        edge = window.requestAnimationFrame(scrollAtEdge);
      };

      const follow = (moved: PointerEvent) => {
        pointer = { x: moved.clientX, y: moved.clientY };
        if (!clone) {
          if (
            touch ||
            fixed ||
            Math.hypot(pointer.x - start.x, pointer.y - start.y) < LIFT_AFTER_PX
          )
            return;
          lift();
        }
        moved.preventDefault();
        place();
        if (frame) return;
        frame = window.requestAnimationFrame(() => {
          frame = 0;
          reconsider();
        });
      };

      const finish = (cancelled: boolean) => {
        stop();
        if (frame) window.cancelAnimationFrame(frame);
        if (edge) window.cancelAnimationFrame(edge);
        delete document.body.dataset.lifting;
        if (!clone) {
          const tapped =
            Math.hypot(pointer.x - start.x, pointer.y - start.y) < TAP_PX;
          if (!cancelled && tapped) onClick?.();
          return;
        }
        if (cancelled || to === from) setOrder(null);
        else {
          latest.current.arrangement.move(block.id, to);
          latest.current.say(
            `“${block.title}” moved to ${to + 1} of ${blocks.length}.`,
          );
        }
        settleClone(clone, block.id, () => setLifted(null));
      };

      const stop = listen(follow, finish);
      if (grip) lift();
    },
    [],
  );
  const pressBlock = useCallback(
    (event: ReactPointerEvent, block: WorkBlock, onClick?: () => void) =>
      press(event, block, onClick, false),
    [press],
  );
  const pressGrip = useCallback(
    (event: ReactPointerEvent, block: WorkBlock) =>
      press(event, block, undefined, true),
    [press],
  );

  const pressEdge = useCallback(
    (event: ReactPointerEvent, block: WorkBlock, startColumn: number) => {
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      const node = document.querySelector<HTMLElement>("[data-block-grid]");
      if (!node) return;
      const grid = node.getBoundingClientRect();
      const measure = (x: number) =>
        snapWidth({
          edge: x,
          gap: BLOCK_GRID_GAP_PX * zoomOf(node),
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
        stop();
        delete document.body.dataset.stretching;
        setStretch(null);
        if (cancelled || width === block.width) return;
        latest.current.writeBlock({ ...block, width });
        latest.current.say(
          `“${block.title}” is now ${WIDTH_LABELS[width].toLowerCase()} width.`,
        );
      };
      const stop = listen(follow, finish);
    },
    [],
  );

  return { lifted, pressBlock, pressEdge, pressGrip, shown, stretch };
}

type Place = { left: number; top: number };

/** useReflow glides every block on the arrange map from where it was to where the new order puts it, starting from wherever a glide in progress has it. */
export function useReflow(key: string, on: boolean) {
  const last = useRef<Map<string, Place> | null>(null);
  // biome-ignore lint/correctness/useExhaustiveDependencies: key names the layout, so the blocks are measured again whenever it changes
  useLayoutEffect(() => {
    if (!on) {
      last.current = null;
      return;
    }
    const before = last.current;
    const next = new Map<string, Place>();
    const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    for (const node of document.querySelectorAll<HTMLElement>(
      "[data-arrange-id]",
    )) {
      const id = node.dataset.arrangeId ?? "";
      const seen = node.getBoundingClientRect();
      for (const glide of node.getAnimations()) glide.cancel();
      const box = node.getBoundingClientRect();
      const place = { left: box.left, top: box.top + window.scrollY };
      next.set(id, place);
      const was = before?.get(id);
      if (!was || still) continue;
      const zoom = zoomOf(node);
      const x = (was.left - place.left + seen.left - box.left) / zoom;
      const y = (was.top - place.top + seen.top - box.top) / zoom;
      if (Math.abs(x) < 1 && Math.abs(y) < 1) continue;
      node.animate([{ translate: `${x}px ${y}px` }, { translate: "0px 0px" }], {
        duration: REFLOW_MS,
        easing: EASE,
      });
    }
    last.current = next;
  }, [key, on]);
}
