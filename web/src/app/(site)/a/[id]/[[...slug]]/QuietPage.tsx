import type { BrowseType } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { emptyPageInvitation } from "@/lib/empty-page-invitation";
import { TYPE_LABELS } from "@/lib/work-types";

export function EmptyPage({ type }: { type: BrowseType }) {
  const label = TYPE_LABELS[type].toLowerCase();

  return (
    <QuietComposition heading="No public content">
      The creator has put none of this {label} on the page. What the file holds
      is kept, and every download carries it.
    </QuietComposition>
  );
}

export function EmptyPageInvitation({
  type,
  coreBlocks,
  canAdd,
}: {
  type: BrowseType;
  coreBlocks: readonly string[];
  canAdd: boolean;
}) {
  return (
    <QuietComposition compact heading="Nothing on this page yet">
      {emptyPageInvitation({
        canAdd,
        coreBlocks,
        typeLabel: TYPE_LABELS[type].toLowerCase(),
      })}
    </QuietComposition>
  );
}

function QuietComposition({
  heading,
  compact = false,
  children,
}: {
  heading: string;
  compact?: boolean;
  children: React.ReactNode;
}) {
  return (
    <section
      aria-labelledby="quiet-page-heading"
      className={cn("rounded-plate bg-inset p-6 md:p-9", compact && "mb-16")}
    >
      <div className="max-w-[42ch]">
        <h2
          className="font-display text-title font-medium tracking-tight text-ink"
          id="quiet-page-heading"
        >
          {heading}
        </h2>
        <p className="mt-3.5 text-ui text-mute">{children}</p>
      </div>
    </section>
  );
}
