"use client";

import { useRouter } from "next/navigation";
import { type RefObject, useCallback, useState } from "react";
import {
  type AssetBlock,
  addAssetBlock,
  arrangeAssetBlocks,
  type ElementType,
  moveAssetBlockContent,
  removeAssetBlock,
} from "@/lib/api/query";
import type { Candidate } from "@/lib/working-copy";
import { arrangementRequest, moveBlock } from "./composition";

export type Arrangement = {
  busy: boolean;
  add: (definition: string, elementType: ElementType) => void;
  move: (blockId: string, to: number) => void;
  remove: (blockId: string) => void;
  moveContent: (blockId: string, destinationBlockId: string) => void;
  setHidden: (blockId: string, hidden: boolean) => void;
};

type Page = {
  assetId: string;
  candidate: Candidate;
  applyServerBlocks: (blocks: AssetBlock[]) => void;
  blocks: RefObject<AssetBlock[]>;
  editBlockList: (change: (blocks: AssetBlock[]) => AssetBlock[]) => void;
  savedBlocks: RefObject<AssetBlock[]>;
  say: (message: string) => void;
};

export function gripId(blockId: string): string {
  return `grip-${blockId}`;
}

function returnFocusToGrip(blockId: string) {
  if (document.activeElement?.id !== gripId(blockId)) return () => {};
  return () =>
    window.requestAnimationFrame(() =>
      window.requestAnimationFrame(() =>
        document.getElementById(gripId(blockId))?.focus(),
      ),
    );
}

export function useArrangement(page: Page): Arrangement {
  const router = useRouter();
  const [busy, setBusy] = useState(false);

  const run = useCallback(
    (action: () => Promise<void>, refusal: string, done?: () => void) => {
      if (busy) return;
      setBusy(true);
      void (async () => {
        try {
          await action();
          router.refresh();
          done?.();
        } catch (error) {
          page.say(error instanceof Error ? error.message : refusal);
        } finally {
          setBusy(false);
        }
      })();
    },
    [busy, page, router],
  );

  const arrange = useCallback(
    (order: AssetBlock[], refusal: string, done?: () => void) =>
      run(
        async () => {
          const saved = await arrangeAssetBlocks(
            page.candidate,
            page.assetId,
            arrangementRequest(order, page.savedBlocks.current),
          );
          page.applyServerBlocks(saved);
        },
        refusal,
        done,
      ),
    [page, run],
  );

  return {
    add: (definition, elementType) =>
      run(async () => {
        const added = await addAssetBlock(
          page.candidate,
          page.assetId,
          definition,
          elementType,
        );
        page.editBlockList((list) => [...list, added]);
        page.say(`${added.title} is on the page.`);
      }, "The block could not be added. Try again."),
    busy,
    move: (blockId, to) => {
      const order = moveBlock(page.blocks.current, blockId, to);
      if (order === page.blocks.current) return;
      arrange(
        order,
        "The block order could not be saved. Try again.",
        returnFocusToGrip(blockId),
      );
    },
    moveContent: (blockId, destinationBlockId) =>
      run(async () => {
        const saved = await moveAssetBlockContent(
          page.candidate,
          page.assetId,
          blockId,
          destinationBlockId,
        );
        page.applyServerBlocks(saved);
      }, "The content could not be moved. Try again."),
    remove: (blockId) =>
      run(async () => {
        await removeAssetBlock(page.candidate, page.assetId, blockId);
        page.editBlockList((list) =>
          list
            .filter((block) => block.id !== blockId)
            .map((block, position) => ({ ...block, position })),
        );
      }, "The block could not be removed. Try again."),
    setHidden: (blockId, hidden) =>
      arrange(
        page.blocks.current.map((block) =>
          block.id === blockId ? { ...block, hidden } : block,
        ),
        "The block could not be changed. Try again.",
      ),
  };
}
