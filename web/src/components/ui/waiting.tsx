import type { ReactNode } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";

/** What a page says while it reads what it needs, in one voice across the site. */
export function Waiting({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <p
      aria-live="polite"
      className={cn(
        "flex items-center gap-2.5 font-ui text-ui text-mute",
        className,
      )}
    >
      <span
        aria-hidden="true"
        className="size-2 shrink-0 animate-pulse rounded-full bg-accent"
      />
      {children}
    </p>
  );
}

/** The same line when it is the whole page, held in the page's own column. */
export function PageWaiting({ children }: { children: ReactNode }) {
  return <Waiting className={`${shellClasses} py-16`}>{children}</Waiting>;
}
