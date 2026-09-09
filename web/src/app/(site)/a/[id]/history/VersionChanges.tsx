"use client";

import Image from "next/image";
import { useEffect, useState } from "react";
import {
  compareAssetVersions,
  type VersionChange,
  type VersionChangeGroup,
  type VersionComparison,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { wordDiff } from "@/lib/text-diff";

const MARKS: Record<VersionChange["kind"], string> = {
  addition: "Added",
  removal: "Removed",
  change: "Changed",
};

const MARK_TONES: Record<VersionChange["kind"], string> = {
  addition: "text-accent",
  removal: "text-stop",
  change: "text-mute",
};

/** What Illarin computed between two recorded versions, apart from anything the creator wrote. */
export function VersionChanges({
  assetId,
  kind,
  from,
  to,
}: {
  assetId: string;
  kind: string;
  from: number;
  to: number;
}) {
  const [compared, setCompared] = useState<VersionComparison | null>(null);
  const [refusal, setRefusal] = useState("");
  const [reading, setReading] = useState(true);

  useEffect(() => {
    let current = true;
    setReading(true);
    void compareAssetVersions(assetId, from, to).then((answer) => {
      if (!current) return;
      setCompared(answer.compared);
      setRefusal(answer.compared ? "" : answer.refusal);
      setReading(false);
    });
    return () => {
      current = false;
    };
  }, [assetId, from, to]);

  if (reading) return <ComparisonSkeleton />;
  if (!compared) return <Refusal>{refusal}</Refusal>;
  if (compared.unavailable) return <Refusal>{compared.unavailable}</Refusal>;

  return (
    <div className="mt-4 max-w-[70ch]">
      {compared.promptsWithheld ? (
        <p className="mb-4 max-w-[60ch] text-meta text-mute">
          This {kind} keeps its prompts for linked applications, so their
          wording is not shown here.
        </p>
      ) : null}

      {compared.groups.length === 0 ? (
        <p className="text-meta text-mute">
          Nothing differs between these two versions.
        </p>
      ) : (
        <div className="grid gap-6">
          {compared.groups.map((group) => (
            <ChangeGroup group={group} key={group.subject} />
          ))}
        </div>
      )}
    </div>
  );
}

/** One subject the comparison found changes in, such as the greetings or the pictures. */
function ChangeGroup({ group }: { group: VersionChangeGroup }) {
  return (
    <section>
      <h4 className="text-meta font-medium text-mute">{group.label}</h4>
      <ul className="mt-1 grid list-none divide-y divide-rule border-rule border-t">
        {group.changes.map((change, index) => (
          <li
            className="py-2.5"
            // Two changes can carry the same name, so their place in the group is the key.
            key={`${group.subject}-${index}`}
          >
            <ChangeRow change={change} subject={group.label} />
          </li>
        ))}
      </ul>
    </section>
  );
}

/** One addition, removal or edit, with its detail behind a disclosure. */
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
            MARK_TONES[change.kind],
          )}
        >
          {MARKS[change.kind]}
        </span>
        <span className="min-w-0 text-ui text-ink">
          {change.name || subject}
        </span>
        {change.previousName ? (
          <span className="text-meta text-mute">was {change.previousName}</span>
        ) : null}
      </p>

      {hasText || hasImage ? (
        <details className="group mt-1 sm:ml-[5.75rem]">
          <summary className="inline-flex min-h-9 cursor-pointer list-none items-center text-meta text-mute outline-offset-3 hover:text-ink">
            <span className="underline decoration-rule underline-offset-4 group-open:decoration-accent">
              {hasImage ? "Show the pictures" : "Show the wording"}
            </span>
          </summary>
          {hasImage ? (
            <Pictures after={change.afterImage} before={change.beforeImage} />
          ) : (
            <Wording after={change.after ?? ""} before={change.before ?? ""} />
          )}
        </details>
      ) : null}
    </>
  );
}

/** The picture a change replaced, beside the one it put there. */
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

/** One text change, with the words an update took out and the words it put in. */
function Wording({ before, after }: { before: string; after: string }) {
  return (
    <p className="mt-2 overflow-x-auto font-prose text-prose break-words whitespace-pre-wrap text-ink">
      {wordDiff(before, after).map((piece, index) => {
        // The pieces have no identity of their own, so their place in the text is the key.
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

/** Why a comparison cannot be shown, in place of the changes rather than beside them. */
function Refusal({ children }: { children: string }) {
  return (
    <p className="mt-4 max-w-[60ch] text-meta text-stop" role="alert">
      {children}
    </p>
  );
}

/** The shape of a comparison while Illarin is still reading it. */
function ComparisonSkeleton() {
  return (
    <div className="mt-4" aria-live="polite">
      <span className="sr-only">Reading the comparison</span>
      <div aria-hidden="true" className="grid gap-2">
        <div className="h-3 w-24 animate-pulse rounded-control bg-deep" />
        <div className="h-10 animate-pulse rounded-control bg-deep" />
        <div className="h-10 w-3/4 animate-pulse rounded-control bg-deep" />
      </div>
    </div>
  );
}
