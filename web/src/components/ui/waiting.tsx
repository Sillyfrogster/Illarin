import type { ReactNode } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";

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

export function PageWaiting({ children }: { children: ReactNode }) {
  return <Waiting className={`${shellClasses} py-16`}>{children}</Waiting>;
}
