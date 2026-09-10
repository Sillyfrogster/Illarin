import type { ComponentPropsWithoutRef, ElementType, ReactNode } from "react";
import { cn } from "@/lib/cn";

export const shellClasses =
  "mx-auto w-full max-w-[var(--shell)] px-[var(--gutter)]";

type ShellProps = ComponentPropsWithoutRef<"div"> & {
  children: ReactNode;
  as?: ElementType;
};

/** Holds page content to a fixed width */
export function Shell({
  children,
  as: Tag = "div",
  className,
  ...rest
}: ShellProps) {
  return (
    <Tag className={cn(shellClasses, className)} {...rest}>
      {children}
    </Tag>
  );
}
