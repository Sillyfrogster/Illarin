import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** A dense working surface whose nested hairlines give it a machined edge, after Cult UI's texture card. */
export function Slab({ className, ...props }: ComponentProps<"section">) {
  return (
    <section
      className={cn(
        "flex min-w-0 flex-col overflow-hidden rounded-plate bg-plane",
        "inset-ring inset-ring-rule",
        "shadow-[inset_0_1px_0_0_var(--v-field),0_1px_2px_-1px_rgb(0_0_0/0.14)]",
        className,
      )}
      {...props}
    />
  );
}

/** The strip that names a slab and carries its one control. */
export function SlabHead({ className, ...props }: ComponentProps<"header">) {
  return (
    <header
      className={cn(
        "flex flex-wrap items-baseline justify-between gap-x-6 gap-y-2 border-b border-rule bg-inset/60 px-4 py-2.5",
        className,
      )}
      {...props}
    />
  );
}

export function SlabTitle({ className, ...props }: ComponentProps<"h2">) {
  return (
    <h2
      className={cn(
        "font-ui text-meta font-medium tracking-[0.06em] text-ink uppercase",
        className,
      )}
      {...props}
    />
  );
}

export function SlabNote({ className, ...props }: ComponentProps<"p">) {
  return (
    <p className={cn("font-ui text-meta text-mute", className)} {...props} />
  );
}

/** The strip under a slab's content, for a legend, a total or a link onward. */
export function SlabFoot({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mt-auto flex flex-wrap items-center justify-between gap-x-6 gap-y-2 border-t border-rule bg-inset/60 px-4 py-2.5 font-ui text-label text-mute",
        className,
      )}
      {...props}
    />
  );
}

/** Horizontal scroll that fades at its far edge, so the cut content reads as travel, after Dice UI's scroller. */
export function SideScroll({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "min-w-0 overflow-x-auto overscroll-x-contain [scrollbar-width:thin]",
        "[mask-image:linear-gradient(to_right,#000_calc(100%-24px),transparent)] sm:[mask-image:none]",
        className,
      )}
      {...props}
    />
  );
}
