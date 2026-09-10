"use client";

import { type DragEvent, useState } from "react";
import type { BlockGrip } from "./BlockTools";

const BLOCK = "text/plain";

export type BlockDrag = {
  dragging: string | null;
  over: string | null;
  grip: (blockId: string) => BlockGrip;
  target: (
    blockId: string,
    position: number,
  ) => {
    onDragLeave: () => void;
    onDragOver: (event: DragEvent) => void;
    onDrop: (event: DragEvent) => void;
  };
};

/** Dragging a block onto another puts it in that block's place. */
export function useBlockDrag(move: (blockId: string, to: number) => void) {
  const [dragging, setDragging] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);

  return {
    dragging,
    grip: (blockId: string) => ({
      draggable: true as const,
      onDragEnd: () => {
        setDragging(null);
        setOver(null);
      },
      onDragStart: (event: DragEvent) => {
        setDragging(blockId);
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData(BLOCK, blockId);
      },
    }),
    over,
    target: (blockId: string, position: number) => ({
      onDragLeave: () =>
        setOver((current) => (current === blockId ? null : current)),
      onDragOver: (event: DragEvent) => {
        if (!event.dataTransfer.types.includes(BLOCK)) return;
        event.preventDefault();
        event.dataTransfer.dropEffect = "move";
        if (dragging !== blockId) setOver(blockId);
      },
      onDrop: (event: DragEvent) => {
        event.preventDefault();
        // The payload outlives the render that started the drag, so it decides.
        const moved = event.dataTransfer.getData(BLOCK) || dragging;
        setDragging(null);
        setOver(null);
        if (moved && moved !== blockId) move(moved, position);
      },
    }),
  };
}
