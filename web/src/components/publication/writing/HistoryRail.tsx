"use client";

import { BookmarkPlus, RotateCcw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  readPostHistory,
  readPostRevisions,
  restorePostRevision,
} from "@/lib/api/posts";
import type { Post, PostAction, PostRevision } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import {
  type HistoryEntry,
  historyStream,
  noteWords,
  revisionWords,
} from "@/lib/post-history";

/**
 * Every edition this post has kept and everything done around them, on one
 * spine. Nothing here reaches readers, and nothing here can be changed.
 */
export function HistoryRail({
  handle,
  keeping,
  post,
  onKeep,
  onRestored,
  onFailure,
}: {
  handle: string;
  keeping: boolean;
  post: Post;
  onKeep: () => void;
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
      onFailure(kept.error ?? "The history could not be read.");
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
      onFailure(answer.error ?? "That edition could not be restored.");
      return;
    }
    onRestored(answer.value);
    void load();
  }

  const kept = stream?.some((entry) => entry.kind === "edition") ?? false;

  return (
    <div className="flex flex-col gap-6">
      <Button loading={keeping} onClick={onKeep} variant="primary">
        <BookmarkPlus aria-hidden="true" />
        Keep this version
      </Button>

      {stream === null ? (
        <p aria-live="polite" className="font-ui text-ui text-mute">
          Opening the history…
        </p>
      ) : (
        <>
          {kept ? null : (
            <p className="font-prose text-meta text-mute">
              Nothing kept yet. Keeping a version puts the writing as it stands
              beyond reach of your next change. Publishing keeps it too.
            </p>
          )}
          <ol className="relative flex list-none flex-col gap-5 border-l border-rule/70 pl-5">
            {stream.map((entry) =>
              entry.kind === "edition" ? (
                <Edition
                  chosen={chosen?.id === entry.revision.id}
                  handle={handle}
                  key={entry.revision.id}
                  onCancel={() => setChosen(null)}
                  onChoose={() => setChosen(entry.revision)}
                  onRestore={() => void restore()}
                  restoring={busy}
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
    </div>
  );
}

function Edition({
  chosen,
  handle,
  onCancel,
  onChoose,
  onRestore,
  restoring,
  revision,
}: {
  chosen: boolean;
  handle: string;
  onCancel: () => void;
  onChoose: () => void;
  onRestore: () => void;
  restoring: boolean;
  revision: PostRevision;
}) {
  return (
    <li className="relative">
      <span
        aria-hidden="true"
        className={cn(
          "absolute top-1.5 -left-[1.4rem] grid size-6 place-items-center rounded-full font-prose text-label font-medium tabular-nums",
          revision.public ? "bg-action text-on-accent" : "bg-deep text-mute",
        )}
      >
        {revision.number}
      </span>
      <div className="rounded-plate bg-deep p-4">
        <p className="font-ui text-meta font-medium text-accent">
          {revisionWords(revision.capturedFor)}
          {revision.public ? " · On the blog now" : ""}
        </p>
        <p className="mt-1 font-display text-ui text-ink wrap-anywhere">
          {revision.title}
        </p>
        <p className="mt-1 flex flex-wrap gap-x-3 font-prose text-meta text-mute">
          <span>{who(revision.capturedBy, handle)}</span>
          <time dateTime={revision.capturedAt}>
            {readableMoment(revision.capturedAt)}
          </time>
        </p>
        {chosen ? (
          <div className="mt-3 flex flex-col gap-3 rounded-control bg-plane p-3">
            <p className="font-prose text-meta text-ink">
              This replaces what you are writing now. Nothing readers have
              changes, no edition is lost, and you publish afterwards to give it
              to them.
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                loading={restoring}
                onClick={onRestore}
                size="compact"
                variant="primary"
              >
                Restore edition {revision.number}
              </Button>
              <Button
                disabled={restoring}
                onClick={onCancel}
                size="compact"
                variant="ghost"
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <Button
            className="mt-3"
            onClick={onChoose}
            size="compact"
            variant="ghost"
          >
            <RotateCcw aria-hidden="true" />
            Restore
          </Button>
        )}
      </div>
    </li>
  );
}

function Note({ action, handle }: { action: PostAction; handle: string }) {
  return (
    <li className="relative">
      <span
        aria-hidden="true"
        className="absolute top-2 -left-[1.15rem] size-2 rounded-full bg-rule"
      />
      <p className="font-prose text-meta text-mute">
        {who(action.actor, handle)} {noteWords(action)} ·{" "}
        <time dateTime={action.at}>{readableMoment(action.at)}</time>
      </p>
    </li>
  );
}

// who names the person who acted, and says you when it was this account.
function who(actor: string, handle: string): string {
  if (!actor) return "A closed account";
  return actor === handle ? "You" : `@${actor}`;
}
