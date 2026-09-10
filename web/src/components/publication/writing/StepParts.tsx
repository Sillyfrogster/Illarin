"use client";

import { Button } from "@/components/ui/button";
import type { Post, PostRevision } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import { revisionWords } from "@/lib/post-history";

export function Heading({ line, title }: { line: string; title: string }) {
  return (
    <div>
      <h3 className="font-display text-section font-medium tracking-tight text-ink">
        {title}
      </h3>
      <p className="mt-2 font-prose text-meta text-mute">{line}</p>
    </div>
  );
}

export function Subject({ post }: { post: Post }) {
  return (
    <p className="rounded-plate bg-deep p-4">
      <span className="block font-display text-ui text-ink wrap-anywhere">
        {post.title || "Untitled post"}
      </span>
      <span className="mt-1 block font-prose text-meta text-mute wrap-anywhere">
        illarin.xyz/blog/{post.slug}
      </span>
    </p>
  );
}

export function Commit({
  busy,
  onCommit,
  ready,
  tone,
  word,
}: {
  busy: boolean;
  onCommit: () => void;
  ready: boolean;
  tone?: "stop";
  word: string;
}) {
  return (
    <Button
      className="self-start"
      disabled={!ready}
      loading={busy}
      onClick={onCommit}
      variant={tone === "stop" ? "stop" : "primary"}
    >
      {word}
    </Button>
  );
}

export function Editions({
  chosen,
  name,
  onChoose,
  revisions,
  standingOf,
}: {
  chosen: string;
  name: string;
  onChoose: (id: string) => void;
  revisions: PostRevision[] | null;
  standingOf: (revision: PostRevision) => string;
}) {
  if (revisions === null) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        Reading the editions…
      </p>
    );
  }
  return (
    <fieldset className="flex flex-col gap-2 border-0">
      <legend className="mb-1 font-ui text-ui text-ink">Edition</legend>
      {revisions.map((one) => {
        const standing = standingOf(one);
        return (
          <label
            className={cn(
              "flex cursor-pointer items-start gap-3 rounded-control p-3",
              chosen === one.id ? "bg-accent-wash" : "bg-deep",
            )}
            key={one.id}
          >
            <input
              checked={chosen === one.id}
              className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
              name={name}
              onChange={() => onChoose(one.id)}
              type="radio"
              value={one.id}
            />
            <span className="min-w-0 flex-1">
              <span className="block font-ui text-ui text-ink">
                {one.number}. {revisionWords(one.capturedFor)}
              </span>
              <span className="block font-prose text-meta text-mute">
                {readableMoment(one.capturedAt)}
              </span>
            </span>
            {standing ? (
              <span className="shrink-0 font-prose text-label text-accent">
                {standing}
              </span>
            ) : null}
          </label>
        );
      })}
    </fieldset>
  );
}
