"use client";

import { Trash2 } from "lucide-react";
import { useState } from "react";
import { recoverPost } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import { standingName } from "@/lib/post-standing";
import styles from "./RecoveryBand.module.css";

export function RecoveryBand({
  post,
  onChanged,
  onFailure,
}: {
  post: Post;
  onChanged: (post: Post) => void;
  onFailure: (message: string) => void;
}) {
  const deletion = post.deletion;
  const [busy, setBusy] = useState(false);

  if (!deletion) return null;

  async function bringBack() {
    setBusy(true);
    const answer = await recoverPost(post.id, post.version);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onChanged(answer.value);
  }

  return (
    <section aria-labelledby="deleted-post" className={styles.band}>
      <p className={styles.mark}>
        <Trash2 size={15} strokeWidth={1.9} aria-hidden="true" />
        <span id="deleted-post">Deleted</span>
      </p>

      <p className={styles.said} suppressHydrationWarning>
        {remainingDeletionWindow(deletion.until)}
      </p>

      <dl className={styles.wording}>
        <dt>Deleted</dt>
        <dd>
          {readableMoment(deletion.at)} by @{deletion.by}
        </dd>
        <dt>Gone for good</dt>
        <dd>
          {readableMoment(deletion.until)}. The writing, every edition and every
          picture go with it.
        </dd>
        <dt>Deleted from</dt>
        <dd>{standingName(post.status)}</dd>
      </dl>

      <div className={styles.actions}>
        <button
          className={styles.act}
          disabled={busy}
          onClick={() => void bringBack()}
          type="button"
        >
          {busy ? "Bringing it back…" : "Bring it back"}
        </button>
      </div>
    </section>
  );
}
