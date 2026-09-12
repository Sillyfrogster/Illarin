import Image from "next/image";
import type { CSSProperties } from "react";
import type { BrowseKind } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { emptyPageInvitation } from "@/lib/empty-page-invitation";
import { KIND_LABELS } from "@/lib/kinds";

export type ArtPlacement = "beside" | "inRow" | "atFoot";

const PLACEMENT: Record<ArtPlacement, string> = {
  beside: "justify-center md:justify-end",
  inRow:
    "justify-center md:justify-end md:[grid-column:var(--block-start)_/_span_var(--block-columns)]",
  atFoot: "justify-end pt-6",
};

export function QuietPageArt({
  placement,
  style,
  compact = false,
}: {
  placement: ArtPlacement;
  style?: CSSProperties;
  compact?: boolean;
}) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "pointer-events-none flex min-w-0 items-start select-none",
        PLACEMENT[placement],
      )}
      data-measurement-ignore
      style={style}
    >
      <Image
        alt=""
        className={cn(
          "h-auto w-full rounded-plate object-contain",
          compact || placement === "atFoot"
            ? "max-w-48"
            : "max-w-60 md:max-w-72",
        )}
        height={1254}
        sizes="(max-width: 767px) 240px, 288px"
        src="/reading/watcher-reading.webp"
        width={1254}
      />
    </div>
  );
}

export function EmptyPage({ kind }: { kind: BrowseKind }) {
  const label = KIND_LABELS[kind].toLowerCase();

  return (
    <QuietComposition heading="No public content">
      The creator has put none of this {label} on the page. What the file holds
      is kept, and every download carries it.
    </QuietComposition>
  );
}

export function EmptyPageInvitation({
  kind,
  coreBlocks,
  canAdd,
}: {
  kind: BrowseKind;
  coreBlocks: readonly string[];
  canAdd: boolean;
}) {
  return (
    <QuietComposition compact heading="Nothing on this page yet">
      {emptyPageInvitation({
        canAdd,
        coreBlocks,
        kindLabel: KIND_LABELS[kind].toLowerCase(),
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
      className={cn(
        "grid items-center gap-y-8 rounded-plate bg-inset p-6 md:gap-x-[clamp(28px,4vw,64px)] md:p-9",
        compact
          ? "mb-16 md:grid-cols-[7fr_5fr]"
          : "md:min-h-[clamp(300px,32vw,420px)] md:grid-cols-[5fr_7fr]",
      )}
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
      <QuietPageArt placement="beside" compact={compact} />
    </section>
  );
}
