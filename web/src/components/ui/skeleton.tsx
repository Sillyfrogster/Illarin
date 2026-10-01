import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** Skeleton holds the place of content whose shape is known while it loads. */
export function Skeleton({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "block rounded-control bg-deep motion-safe:animate-pulse",
        className,
      )}
      data-slot="skeleton"
      {...props}
    />
  );
}
