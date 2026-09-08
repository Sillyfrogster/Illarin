"use client";

import { useEffect, useRef } from "react";
import type { AssetElement } from "@/lib/api/query";
import styles from "./UnsealConfirmation.module.css";

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
      ref={dialog}
      className={styles.dialog}
      onCancel={(event) => {
        event.preventDefault();
        onKeepSealed();
      }}
    >
      <div className={styles.body}>
        <h2>Make this prompt public?</h2>
        <p>
          Saving puts the text of {namePrompts(prompts)} in front of every
          reader, in this version and in every recorded one.
          {keepsASeal ? "" : " Ordinary downloads come back with it."}
        </p>
      </div>
      <footer className={styles.footer}>
        <button type="button" disabled={pending} onClick={onKeepSealed}>
          Keep sealed
        </button>
        <button
          type="button"
          className={styles.expose}
          disabled={pending}
          onClick={onExpose}
        >
          Make prompt public
        </button>
      </footer>
    </dialog>
  );
}

/** Writes a list of prompts the way a person reads one. */
function namePrompts(names: string[]): string {
  if (names.length < 2) return names[0] ?? "this prompt";
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}
