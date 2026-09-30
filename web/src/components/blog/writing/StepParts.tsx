"use client";

import { Button } from "@/components/ui/button";
import { RadioGroup } from "@/components/ui/radio-group";
import { ShinyButton } from "@/components/ui/shiny-button";
import type { PostRevision } from "@/lib/api/query";
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
  tone?: "stop" | "publish";
  word: string;
}) {
  if (tone === "publish")
    return (
      <ShinyButton
        className="self-start"
        disabled={!ready}
        loading={busy}
        onClick={onCommit}
      >
        {word}
      </ShinyButton>
    );
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
        Loading revisions…
      </p>
    );
  }
  return (
    <fieldset className="min-w-0 border-0">
      <legend className="mb-1 font-ui text-ui text-ink">Revision</legend>
      <RadioGroup
        name={name}
        onValueChange={onChoose}
        options={revisions.map((one) => {
          const standing = standingOf(one);
          return {
            value: one.id,
            label: `${one.number}. ${revisionWords(one.capturedFor)}`,
            hint: (
              <>
                {readableMoment(one.capturedAt)}
                {standing ? (
                  <span className="text-accent"> · {standing}</span>
                ) : null}
              </>
            ),
          };
        })}
        value={chosen}
      />
    </fieldset>
  );
}
