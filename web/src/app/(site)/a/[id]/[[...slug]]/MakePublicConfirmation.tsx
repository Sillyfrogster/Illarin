"use client";

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
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
  return (
    <AlertDialog
      onOpenChange={(open) => {
        if (!open) onKeepPrivate();
      }}
      open
    >
      <AlertDialogContent>
        <AlertDialogTitle>Make these prompts public?</AlertDialogTitle>
        <AlertDialogDescription>
          {replacement ? "Applying this file" : "Saving"} makes{" "}
          {namePrompts(prompts)} public. Readers can read them at once, in this
          version and earlier ones.
          {keepsAPrivatePrompt
            ? ""
            : " With no private prompts left, this preset downloads as a file again."}
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel asChild>
            <Button disabled={pending} variant="ghost">
              Keep private
            </Button>
          </AlertDialogCancel>
          <Button loading={pending} onClick={onMakePublic} variant="stop">
            {replacement ? "Apply and make public" : "Make public"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function namePrompts(names: string[]): string {
  if (names.length < 2) return names[0] ?? "this prompt";
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}
