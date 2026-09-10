import type { ReactNode } from "react";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";

/** One group of an administration page, with what it holds and what can be started from it. */
export function Section({
  action,
  children,
  count,
  id,
  lead,
  retired,
  retiredLabel,
  title,
}: {
  action?: ReactNode;
  children: ReactNode;
  count?: number;
  id?: string;
  /** Anything before the title, such as a mark for the thing this section is about. */
  lead?: ReactNode;
  retired?: ReactNode;
  retiredLabel?: string;
  title: string;
}) {
  return (
    <section className="min-w-0" id={id}>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h2 className="flex min-w-0 items-center gap-3 font-display text-section font-medium tracking-tight text-ink">
          {lead}
          <span className="min-w-0 wrap-anywhere">{title}</span>
          {count === undefined ? null : (
            <span className="font-prose text-meta text-mute tabular-nums">
              {count}
            </span>
          )}
        </h2>
        {action}
      </div>
      <div className="mt-5 min-w-0">{children}</div>
      {retired && retiredLabel ? (
        <div className="mt-6">
          <MorphingDisclosure
            className="rounded-plate bg-deep px-5 py-4"
            summary={retiredLabel}
          >
            <div className="mt-4">{retired}</div>
          </MorphingDisclosure>
        </div>
      ) : null}
    </section>
  );
}
