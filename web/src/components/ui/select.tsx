import { ChevronDown } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

export function Select({
  className,
  children,
  ...rest
}: ComponentProps<"select">) {
  return (
    <span className="relative inline-grid max-w-full items-center">
      <select
        className={cn(
          "col-start-1 row-start-1 min-h-11 w-full appearance-none rounded-control border-0 bg-deep py-2 pr-9 pl-3",
          "text-ui text-ink outline-offset-3 disabled:opacity-60",
          className,
        )}
        {...rest}
      >
        {children}
      </select>
      <ChevronDown
        aria-hidden="true"
        className="pointer-events-none col-start-1 row-start-1 mr-3 size-4 justify-self-end text-mute"
      />
    </span>
  );
}
