import type { CSSProperties } from "react";
import type { BrowseKind } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { emptyPageInvitation } from "@/lib/empty-page-invitation";
import { KIND_LABELS } from "@/lib/kinds";
import { quietPageArtVariables } from "@/lib/quiet-page-art";

export type ArtPlacement = "beside" | "inRow" | "atFoot";

const ART = [
  "relative [--art-bleed:max(72px,(100vw-var(--shell))/2+var(--gutter))]",
  "before:absolute before:-z-1 before:bg-[image:var(--quiet-art-light)] before:bg-cover before:bg-[position:center_42%] before:bg-no-repeat before:opacity-90 before:content-['']",
  "before:[mask-composite:intersect] before:[mask-image:linear-gradient(to_right,transparent,#000_34%),linear-gradient(to_bottom,transparent,#000_22%,#000_74%,transparent)]",
  "dark:before:bg-[image:var(--quiet-art-dark)] dark:before:opacity-85",
].join(" ");

const PLACEMENT: Record<ArtPlacement, string> = {
  beside:
    "self-stretch min-h-[clamp(200px,46vw,300px)] md:min-h-[clamp(260px,28vw,380px)] before:inset-y-0 before:left-0 before:w-[calc(100%+var(--art-bleed))]",
  inRow:
    "min-h-52 md:min-h-65 md:[grid-column:var(--block-start)_/_span_var(--block-columns)] before:inset-y-0 before:left-[10%] before:w-[calc(90%+var(--gutter))] md:before:-inset-y-13 md:before:left-0 md:before:w-[calc(100%+var(--art-bleed))]",
  atFoot:
    "h-52 md:mt-12 md:h-75 before:inset-y-0 before:left-[10%] before:w-[calc(90%+var(--gutter))] md:before:left-[42%] md:before:w-[calc(58%+var(--art-bleed))]",
};

export function QuietPageArt({
  kind,
  placement,
  style,
  compact = false,
}: {
  kind: BrowseKind;
  placement: ArtPlacement;
  style?: CSSProperties;
  compact?: boolean;
}) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        ART,
        PLACEMENT[placement],
        compact &&
          "min-h-[clamp(190px,18vw,240px)] md:min-h-[clamp(190px,18vw,240px)]",
      )}
      data-measurement-ignore
      style={{ ...quietPageArtVariables(kind), ...style } as CSSProperties}
    />
  );
}

export function EmptyPage({ kind }: { kind: BrowseKind }) {
  const label = KIND_LABELS[kind].toLowerCase();

  return (
    <QuietComposition heading="No public content" kind={kind}>
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
    <QuietComposition compact heading="Nothing on this page yet" kind={kind}>
      {emptyPageInvitation({
        canAdd,
        coreBlocks,
        kindLabel: KIND_LABELS[kind].toLowerCase(),
      })}
    </QuietComposition>
  );
}

function QuietComposition({
  kind,
  heading,
  compact = false,
  children,
}: {
  kind: BrowseKind;
  heading: string;
  compact?: boolean;
  children: React.ReactNode;
}) {
  return (
    <section
      aria-labelledby="quiet-page-heading"
      className={cn(
        "grid items-center gap-y-8 md:gap-x-[clamp(28px,4vw,64px)]",
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
      <QuietPageArt kind={kind} placement="beside" compact={compact} />
    </section>
  );
}
