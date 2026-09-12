"use client";

import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import type { AssetElement } from "@/lib/api/query";

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

export function UnsealConfirmation({
  prompts,
  keepsASeal,
  pending,
  onKeepSealed,
  onExpose,
  replacement = false,
}: {
  prompts: string[];
  keepsASeal: boolean;
  pending: boolean;
  onKeepSealed: () => void;
  onExpose: () => void;
  replacement?: boolean;
}) {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => dialog.current?.showModal(), []);

  return (
    <dialog
      aria-labelledby="unseal-title"
      aria-describedby="unseal-description"
      className="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-plate bg-plane p-0 text-ink backdrop:bg-black/60"
      onCancel={(event) => {
        event.preventDefault();
        onKeepSealed();
      }}
      ref={dialog}
    >
      <div className="p-6">
        <h2
          id="unseal-title"
          className="font-display text-section font-medium text-ink"
        >
          Remove prompt protection?
        </h2>
        <p id="unseal-description" className="mt-3 text-ui text-mute">
          {replacement ? "Applying this file" : "Saving"} removes protection
          from {namePrompts(prompts)}. Text in published and recorded versions
          can become readable immediately. New draft writing stays private until
          you publish.
          {keepsASeal
            ? ""
            : " If no sealed prompts remain, file downloads become available again."}
        </p>
      </div>
      <footer className="flex flex-wrap justify-end gap-2 border-rule border-t p-4">
        <Button disabled={pending} onClick={onKeepSealed} variant="ghost">
          Keep sealed
        </Button>
        <Button loading={pending} onClick={onExpose} variant="stop">
          {replacement ? "Apply and remove protection" : "Remove protection"}
        </Button>
      </footer>
    </dialog>
  );
}

function namePrompts(names: string[]): string {
  if (names.length < 2) return names[0] ?? "this prompt";
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}
