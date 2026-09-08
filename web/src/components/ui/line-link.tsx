import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/cn";

type LineLinkProps = Omit<ComponentProps<typeof Link>, "children"> & {
  children: ReactNode;
  current?: boolean;
};

/** A link whose rule draws in from the left and retracts to the right */
export function LineLink({
  children,
  className,
  current,
  ...props
}: LineLinkProps) {
  return (
    <Link
      aria-current={current ? "page" : undefined}
      className={cn(
        "group flex min-h-11 items-center text-ui font-medium whitespace-nowrap text-mute transition-colors hover:text-ink aria-[current=page]:text-ink",
        className,
      )}
      {...props}
    >
      <span
        className={cn(
          "relative before:absolute before:inset-x-0 before:top-[calc(100%+3px)] before:h-px before:origin-right before:scale-x-0 before:bg-accent before:transition-transform before:duration-300 before:content-['']",
          "group-hover:before:origin-left group-hover:before:scale-x-100 group-focus-visible:before:origin-left group-focus-visible:before:scale-x-100",
          "group-aria-[current=page]:before:origin-left group-aria-[current=page]:before:scale-x-100 motion-reduce:before:transition-none",
        )}
      >
        {children}
      </span>
    </Link>
  );
}
