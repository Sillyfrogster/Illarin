"use client";

import { RotateCcw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { FormDialog } from "@/components/console/FormDialog";
import {
  readPostHistory,
  readPostRevisions,
  restorePostRevision,
} from "@/lib/api/posts";
import type { Post, PostAction, PostRevision } from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import {
  type HistoryEntry,
  historyStream,
  noteWords,
  revisionWords,
} from "@/lib/post-history";
import styles from "./PostHistory.module.css";

export function PostHistory({
  handle,
  post,
  onRestored,
  onFailure,
}: {
  handle: string;
  post: Post;
  onRestored: (post: Post) => void;
  onFailure: (message: string) => void;
}) {
  const [stream, setStream] = useState<HistoryEntry[] | null>(null);
  const [chosen, setChosen] = useState<PostRevision | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const [kept, done] = await Promise.all([
      readPostRevisions(post.id),
      readPostHistory(post.id),
    ]);
    if (kept.error || !kept.value) {
      onFailure(kept.error ?? "");
      return;
    }
    setStream(historyStream(kept.value.revisions, done.value?.actions ?? []));
  }, [post.id, onFailure]);

  useEffect(() => {
    void load();
  }, [load]);

  async function restore() {
    if (!chosen) return;
    setBusy(true);
    const answer = await restorePostRevision(post.id, chosen.id, post.version);
    setBusy(false);
    setChosen(null);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onRestored(answer.value);
    void load();
  }

  return (
    <section aria-labelledby="post-history" className={styles.history}>
      <h2 className={styles.title} id="post-history">
        Editorial history
      </h2>
      <p className={styles.lead}>
        Every edition this post has kept, and what has been done around them.
        Nothing here can change. Restoring an edition copies it forward into
        what you are writing.
      </p>

      {stream === null ? (
        <p className={styles.waiting}>Opening the history…</p>
      ) : (
        <>
          {stream.some((entry) => entry.kind === "edition") ? null : (
            <p className={styles.empty}>
              Nothing kept yet. Keep this version to put the writing as it
              stands beyond reach of your next change, or publish to keep it and
              give it to readers.
            </p>
          )}
          <ol className={styles.stream}>
            {stream.map((entry) =>
              entry.kind === "edition" ? (
                <Edition
                  handle={handle}
                  key={entry.revision.id}
                  onRestore={() => setChosen(entry.revision)}
                  revision={entry.revision}
                />
              ) : (
                <Note
                  action={entry.action}
                  handle={handle}
                  key={entry.action.id}
                />
              ),
            )}
          </ol>
        </>
      )}

      <FormDialog
        busy={busy}
        commit={chosen ? `Restore edition ${chosen.number}` : "Restore"}
        hint="Nothing readers have changes, and no edition is lost."
        onClose={() => setChosen(null)}
        onCommit={() => void restore()}
        open={chosen !== null}
        title="Put this edition back?"
      >
        <p className={styles.confirm}>
          {chosen?.title}
          <span>
            It replaces what you are writing now. Publish afterwards to give it
            to readers.
          </span>
        </p>
      </FormDialog>
    </section>
  );
}

function Edition({
  handle,
  revision,
  onRestore,
}: {
  handle: string;
  revision: PostRevision;
  onRestore: () => void;
}) {
  return (
    <li className={styles.edition} data-public={revision.public || undefined}>
      <p className={styles.mark}>
        <span className={styles.number}>{revision.number}</span>
        {revisionWords(revision.capturedFor)}
      </p>
      <p className={styles.edited}>{revision.title}</p>
      <p className={styles.who}>
        <span>{who(revision.capturedBy, handle)}</span>
        <time dateTime={revision.capturedAt}>
          {readableMoment(revision.capturedAt)}
        </time>
        {revision.public ? (
          <span className={styles.live}>On the blog now</span>
        ) : null}
      </p>
      <button className={styles.restore} onClick={onRestore} type="button">
        <RotateCcw size={14} strokeWidth={1.8} aria-hidden="true" />
        Restore
      </button>
    </li>
  );
}

function Note({ action, handle }: { action: PostAction; handle: string }) {
  return (
    <li className={styles.note}>
      <span className={styles.said}>
        {who(action.actor, handle)} {noteWords(action)}
      </span>
      <time dateTime={action.at}>{readableMoment(action.at)}</time>
    </li>
  );
}

// who names the person who acted, and says you when it was this account.
function who(actor: string, handle: string): string {
  if (!actor) return "A closed account";
  return actor === handle ? "You" : `@${actor}`;
}
