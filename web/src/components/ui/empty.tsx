import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** Empty is a page or panel with nothing in it: a centred title, a line of help and what to do next. */
function Empty({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex min-w-0 flex-1 flex-col items-center justify-center gap-6 rounded-card p-6 text-center text-balance md:p-12",
        className,
      )}
      data-slot="empty"
      {...props}
    />
  );
}

function EmptyHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("flex max-w-md flex-col items-center gap-2", className)}
      data-slot="empty-header"
      {...props}
    />
  );
}

function EmptyMedia({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mb-2 flex size-12 shrink-0 items-center justify-center rounded-art bg-accent-wash text-accent [&_svg]:size-6 [&_svg]:shrink-0",
        className,
      )}
      data-slot="empty-media"
      {...props}
    />
  );
}

function EmptyTitle({
  as: Tag = "h2",
  className,
  ...props
}: ComponentProps<"h2"> & { as?: "h1" | "h2" | "h3" }) {
  return (
    <Tag
      className={cn(
        "font-display text-title font-medium tracking-[-0.02em] text-ink",
        className,
      )}
      data-slot="empty-title"
      {...props}
    />
  );
}

function EmptyDescription({ className, ...props }: ComponentProps<"p">) {
  return (
    <p
      className={cn(
        "max-w-[46ch] font-prose text-prose text-mute [&>a]:text-accent [&>a]:underline-offset-4 [&>a:hover]:underline",
        className,
      )}
      data-slot="empty-description"
      {...props}
    />
  );
}

function EmptyContent({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex w-full min-w-0 flex-wrap items-center justify-center gap-3",
        className,
      )}
      data-slot="empty-content"
      {...props}
    />
  );
}

export {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
};
