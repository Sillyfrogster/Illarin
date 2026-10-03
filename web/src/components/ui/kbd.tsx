import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** Kbd is a key hint on a grey chip. */
export function Kbd({ className, ...props }: ComponentProps<"kbd">) {
  return (
    <kbd
      className={cn(
        "pointer-events-none inline-flex h-5 min-w-5 items-center justify-center gap-1 rounded-chip bg-fill px-1 font-ui text-label font-medium text-mute select-none [&_svg]:size-3",
        className,
      )}
      {...props}
    />
  );
}
