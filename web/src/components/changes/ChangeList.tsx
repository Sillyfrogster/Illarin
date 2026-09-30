"use client";

import { ChevronRight } from "lucide-react";
import Image from "next/image";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import type { VersionChange, VersionChangeGroup } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { wordDiff } from "@/lib/text-diff";

const MARKS: Record<VersionChange["type"], string> = {
  addition: "Added",
  removal: "Removed",
  change: "Changed",
};

const MARK_TONES: Record<VersionChange["type"], string> = {
  addition: "text-accent",
  removal: "text-stop",
  change: "text-mute",
};

export function ChangeList({ groups }: { groups: VersionChangeGroup[] }) {
  return (
    <div className="grid gap-6">
      {groups.map((group) => (
        <ChangeGroup group={group} key={group.subject} />
      ))}
    </div>
  );
}

function ChangeGroup({ group }: { group: VersionChangeGroup }) {
  return (
    <section>
      <h4 className="text-meta font-medium text-mute">{group.label}</h4>
      <ul className="mt-1 grid list-none divide-y divide-rule border-rule border-t">
        {group.changes.map((change, index) => (
          <li className="py-2.5" key={`${group.subject}-${index}`}>
            <ChangeRow change={change} subject={group.label} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function ChangeRow({
  change,
  subject,
}: {
  change: VersionChange;
  subject: string;
}) {
  const hasText = Boolean(change.before || change.after);
  const hasImage = Boolean(change.beforeImage || change.afterImage);

  return (
    <>
      <p className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span
          className={cn(
            "w-[4.75rem] shrink-0 text-meta font-medium",
            MARK_TONES[change.type],
          )}
        >
          {MARKS[change.type]}
        </span>
        <span className="min-w-0 text-ui text-ink">
          {change.name || subject}
        </span>
        {change.previousName ? (
          <span className="text-meta text-mute">was {change.previousName}</span>
        ) : null}
        {change.note ? (
          <span className="text-meta text-mute">{change.note}</span>
        ) : null}
      </p>

      {hasText || hasImage ? (
        <Collapsible className="mt-1 sm:ml-[5.75rem]">
          <CollapsibleTrigger asChild>
            <Button
              className="group/change -ml-2 px-2"
              size="compact"
              variant="ghost"
            >
              <ChevronRight
                aria-hidden="true"
                className="transition-transform duration-80 group-data-[state=open]/change:rotate-90 motion-reduce:transition-none"
              />
              {hasImage ? "Show the images" : "Show the wording"}
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            {hasImage ? (
              <Pictures after={change.afterImage} before={change.beforeImage} />
            ) : (
              <Wording
                after={change.after ?? ""}
                before={change.before ?? ""}
              />
            )}
          </CollapsibleContent>
        </Collapsible>
      ) : null}
    </>
  );
}

function Pictures({ before, after }: { before?: string; after?: string }) {
  return (
    <div className="mt-3 flex flex-wrap gap-4">
      {before ? <Picture address={before} words="Before" /> : null}
      {after ? <Picture address={after} words="After" /> : null}
    </div>
  );
}

function Picture({ address, words }: { address: string; words: string }) {
  return (
    <figure className="grid justify-items-start gap-2">
      <Image
        alt=""
        className="max-h-40 w-auto rounded-control bg-media"
        height={160}
        src={address}
        unoptimized
        width={160}
      />
      <figcaption className="text-meta text-mute">{words}</figcaption>
    </figure>
  );
}

function Wording({ before, after }: { before: string; after: string }) {
  return (
    <p className="mt-2 overflow-x-auto font-prose text-prose break-words whitespace-pre-wrap text-ink">
      {wordDiff(before, after).map((piece, index) => {
        const key = `${index}-${piece.kind}`;
        if (piece.kind === "removed") {
          return (
            <del
              className="mx-px rounded-[3px] bg-stop-wash px-1 text-stop decoration-stop"
              key={key}
            >
              {piece.text}
            </del>
          );
        }
        if (piece.kind === "added") {
          return (
            <ins
              className="mx-px rounded-[3px] bg-accent-wash px-1 text-ink decoration-accent underline-offset-[3px]"
              key={key}
            >
              {piece.text}
            </ins>
          );
        }
        return <span key={key}>{piece.text}</span>;
      })}
    </p>
  );
}
