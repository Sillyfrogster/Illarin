"use client";

import {
  type DragEndEvent,
  type DragMoveEvent,
  type DragStartEvent,
  MouseSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { useCallback, useEffect, useRef, useState } from "react";
import type { ShelfPiece } from "@/lib/api/query";
import type { Flight } from "./ShelfFlight";
import { positionAt } from "./shelf-places";

/** Where a dragged section lands: a new block at a position, or the end of a text element. */
export type ShelfTarget =
  | { position: number }
  | { elementId: string; blockId: string };

/** Where a dropped piece flies from and to, when both ends are on screen. */
export type DropPath = Pick<Flight, "from" | "to"> | null;

const EDGE_PX = 110;
const EDGE_SPEED = 24;

/** aimAt reads what sits under the pointer: a text element, the ghost, a block, or the page between blocks. */
function aimAt(
  x: number,
  y: number,
  last: ShelfTarget | null,
): ShelfTarget | null {
  for (const node of document.elementsFromPoint(x, y)) {
    if (!(node instanceof HTMLElement)) continue;
    const { shelfBlock, shelfGhost, shelfIndex, shelfPage, shelfText } =
      node.dataset;
    if (shelfText) return { elementId: shelfText, blockId: shelfBlock ?? "" };
    if (shelfGhost !== undefined) return last;
    if (shelfIndex !== undefined) {
      const rect = node.getBoundingClientRect();
      return { position: positionAt(Number(shelfIndex), y, rect) };
    }
    if (shelfPage !== undefined) return last ?? { position: Number(shelfPage) };
  }
  return null;
}

function sameTarget(one: ShelfTarget | null, other: ShelfTarget | null) {
  if (one === null || other === null) return one === other;
  if ("position" in one) {
    return "position" in other && one.position === other.position;
  }
  return "elementId" in other && one.elementId === other.elementId;
}

function landingRect(landing: ShelfTarget): DOMRect | undefined {
  const spot =
    "position" in landing
      ? document.querySelector("[data-shelf-ghost]")
      : document.querySelector(`[data-shelf-text="${landing.elementId}"]`);
  return spot?.getBoundingClientRect();
}

/** useShelfDrag follows a dragged piece, aims it at the page under the pointer, and scrolls the window near its edges. */
export function useShelfDrag({
  onStart,
  onDrop,
}: {
  onStart: () => void;
  onDrop: (piece: ShelfPiece, landing: ShelfTarget, path: DropPath) => void;
}) {
  const [dragging, setDragging] = useState<ShelfPiece | null>(null);
  const [target, setTarget] = useState<ShelfTarget | null>(null);
  const pointer = useRef<{ x: number; y: number } | null>(null);
  const aimed = useRef<ShelfTarget | null>(null);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
  );

  const aim = useCallback(() => {
    const at = pointer.current;
    if (!at) return;
    const next = aimAt(at.x, at.y, aimed.current);
    if (sameTarget(next, aimed.current)) return;
    aimed.current = next;
    setTarget(next);
  }, []);

  // Near the top or bottom of the window a drag scrolls the page, since the rail it starts in is fixed
  useEffect(() => {
    if (!dragging) return;
    let frame = 0;
    const tick = () => {
      const at = pointer.current;
      const low = window.innerHeight - EDGE_PX;
      const edge = !at
        ? 0
        : at.y < EDGE_PX
          ? at.y - EDGE_PX
          : at.y > low
            ? at.y - low
            : 0;
      if (edge !== 0) {
        window.scrollBy(0, (edge / EDGE_PX) * EDGE_SPEED);
        aim();
      }
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [dragging, aim]);

  function stop() {
    setDragging(null);
    setTarget(null);
    aimed.current = null;
    pointer.current = null;
  }

  return {
    dragging,
    target,
    context: {
      autoScroll: false,
      sensors,
      onDragCancel: stop,
      onDragStart: (event: DragStartEvent) => {
        const piece = event.active.data.current?.piece as
          | ShelfPiece
          | undefined;
        setDragging(piece ?? null);
        onStart();
      },
      onDragMove: (event: DragMoveEvent) => {
        const start = event.activatorEvent as MouseEvent;
        pointer.current = {
          x: start.clientX + event.delta.x,
          y: start.clientY + event.delta.y,
        };
        aim();
      },
      onDragEnd: (event: DragEndEvent) => {
        const piece = dragging;
        const landing = aimed.current;
        const from = event.active.rect.current.translated;
        stop();
        if (!piece || !landing) return;
        const to = landingRect(landing);
        onDrop(piece, landing, from && to ? { from, to } : null);
      },
    },
  };
}
