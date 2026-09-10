"use client";

import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import type { AssetElement } from "@/lib/api/query";

/** The sealed prompts this edit turns public, named as the creator wrote them. */
export function unsealedPrompts(
  previous: AssetElement,
  next: AssetElement,
): string[] {
  if (
    previous.type !== "prompt_list" ||
    next.type !== "prompt_list" ||
    !("fragments" in previous.content) ||
    !("fragments" in next.content)
  ) {
    return [];
  }
  const wasSealed = new Set(
    previous.content.fragments
      .filter((fragment) => fragment.protected)
      .map((fragment) => fragment.id),
  );
  return next.content.fragments
    .filter((fragment) => wasSealed.has(fragment.id) && !fragment.protected)
    .map((fragment) => fragment.name || "an untitled prompt");
}

/** Asks before a save puts sealed prompt text in front of every reader. */
export function UnsealConfirmation({
  prompts,
  keepsASeal,
  pending,
  onKeepSealed,
  onExpose,
}: {
  prompts: string[];
  keepsASeal: boolean;
  pending: boolean;
  onKeepSealed: () => void;
  onExpose: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => dialog.current?.showModal(), []);

  return (
    <dialog
      className="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-plate bg-plane p-0 text-ink backdrop:bg-black/60"
      onCancel={(event) => {
        event.preventDefault();
        onKeepSealed();
      }}
      ref={dialog}
    >
      <div className="p-6">
        <h2 className="font-display text-section font-medium text-ink">
          Make this prompt public?
        </h2>
        <p className="mt-3 text-ui text-mute">
          Saving puts the text of {namePrompts(prompts)} in front of every
          reader, in this version and in every recorded one.
          {keepsASeal ? "" : " Ordinary downloads come back with it."}
        </p>
      </div>
      <footer className="flex flex-wrap justify-end gap-2 border-rule border-t p-4">
        <Button disabled={pending} onClick={onKeepSealed} variant="ghost">
          Keep sealed
        </Button>
        <Button loading={pending} onClick={onExpose} variant="stop">
          Make prompt public
        </Button>
      </footer>
    </dialog>
  );
}

/** Writes a list of prompts the way a person reads one. */
function namePrompts(names: string[]): string {
  if (names.length < 2) return names[0] ?? "this prompt";
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}
