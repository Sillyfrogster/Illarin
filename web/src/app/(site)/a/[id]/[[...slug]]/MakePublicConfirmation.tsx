"use client";

import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import type { WorkElement } from "@/lib/api/query";

export function promptsMadePublic(
  previous: WorkElement,
  next: WorkElement,
): string[] {
  if (
    previous.type !== "prompt_list" ||
    next.type !== "prompt_list" ||
    !("fragments" in previous.content) ||
    !("fragments" in next.content)
  ) {
    return [];
  }
  const wasPrivate = new Set(
    previous.content.fragments
      .filter((fragment) => fragment.private)
      .map((fragment) => fragment.id),
  );
  return next.content.fragments
    .filter((fragment) => wasPrivate.has(fragment.id) && !fragment.private)
    .map((fragment) => fragment.name || "an untitled prompt");
}

export function MakePublicConfirmation({
  prompts,
  keepsAPrivatePrompt,
  pending,
  onKeepPrivate,
  onMakePublic,
  replacement = false,
}: {
  prompts: string[];
  keepsAPrivatePrompt: boolean;
  pending: boolean;
  onKeepPrivate: () => void;
  onMakePublic: () => void;
  replacement?: boolean;
}) {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => dialog.current?.showModal(), []);

  return (
    <dialog
      aria-labelledby="make-public-title"
      aria-describedby="make-public-description"
      className="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-plate bg-plane p-0 text-ink backdrop:bg-black/60"
      onCancel={(event) => {
        event.preventDefault();
        onKeepPrivate();
      }}
      ref={dialog}
    >
      <div className="p-6">
        <h2
          id="make-public-title"
          className="font-display text-section font-medium text-ink"
        >
          Make these prompts public?
        </h2>
        <p id="make-public-description" className="mt-3 text-ui text-mute">
          {replacement ? "Applying this file" : "Saving"} makes{" "}
          {namePrompts(prompts)} public. Text in published and recorded versions
          can become readable immediately. New draft writing stays private until
          you publish.
          {keepsAPrivatePrompt
            ? ""
            : " If no private prompts remain, file downloads become available again."}
        </p>
      </div>
      <footer className="flex flex-wrap justify-end gap-2 border-rule border-t p-4">
        <Button disabled={pending} onClick={onKeepPrivate} variant="ghost">
          Keep private
        </Button>
        <Button loading={pending} onClick={onMakePublic} variant="stop">
          {replacement ? "Apply and make public" : "Make public"}
        </Button>
      </footer>
    </dialog>
  );
}

function namePrompts(names: string[]): string {
  if (names.length < 2) return names[0] ?? "this prompt";
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}
