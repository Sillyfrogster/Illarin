import { Slot } from "@radix-ui/react-slot";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** Lists entries down one line, in the order given. */
function Timeline({ className, ...props }: ComponentProps<"ol">) {
  return (
    <ol
      className={cn(
        "relative flex list-none flex-col p-0 [--timeline-connector-thickness:2px] [--timeline-dot-offset:0] [--timeline-dot-size:0.875rem]",
        className,
      )}
      data-slot="timeline"
      {...props}
    />
  );
}

function TimelineItem({ className, ...props }: ComponentProps<"li">) {
  return (
    <li
      className={cn(
        "group/timeline-item relative flex gap-3 pb-2 last:pb-0",
        className,
      )}
      data-slot="timeline-item"
      {...props}
    />
  );
}

function TimelineDot({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "relative z-10 mt-[var(--timeline-dot-offset)] flex size-[var(--timeline-dot-size)] shrink-0 items-center justify-center rounded-full border-2 border-edge bg-plane",
        className,
      )}
      data-slot="timeline-dot"
      {...props}
    />
  );
}

function TimelineConnector({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "absolute start-[calc(var(--timeline-dot-size)/2-var(--timeline-connector-thickness)/2)] top-[calc(var(--timeline-dot-offset)+var(--timeline-dot-size)+0.25rem)] -bottom-[calc(var(--timeline-dot-offset)-0.25rem)] w-[var(--timeline-connector-thickness)] rounded-full bg-rule group-last/timeline-item:hidden",
        className,
      )}
      data-slot="timeline-connector"
      {...props}
    />
  );
}

function TimelineContent({
  asChild = false,
  className,
  ...props
}: ComponentProps<"div"> & { asChild?: boolean }) {
  const Content = asChild ? Slot : "div";
  return (
    <Content
      className={cn("min-w-0 flex-1", className)}
      data-slot="timeline-content"
      {...props}
    />
  );
}

export {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineItem,
};
