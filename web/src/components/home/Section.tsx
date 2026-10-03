import type { ReactNode } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";

/** Section is one band of the landing page: a heading, an optional line under it, and what it shows. */
export function Section({
  action,
  children,
  className,
  id,
  line,
  title,
}: {
  action?: ReactNode;
  children: ReactNode;
  className?: string;
  id: string;
  line?: ReactNode;
  title: string;
}) {
  return (
    <section
      aria-labelledby={id}
      className={cn(shellClasses, "pt-chapter", className)}
    >
      <div className="flex flex-wrap items-end justify-between gap-x-8 gap-y-4">
        <div className="max-w-[40rem]">
          <h2
            className="font-display text-title font-medium tracking-tight text-ink"
            id={id}
          >
            {title}
          </h2>
          {line ? <p className="mt-2 text-lede text-mute">{line}</p> : null}
        </div>
        {action}
      </div>
      <div className="mt-group">{children}</div>
    </section>
  );
}

/** Split is a band with its words on one side and a picture of the feature on the other. */
export function Split({
  children,
  id,
  reverse = false,
  title,
  visual,
}: {
  children: ReactNode;
  id: string;
  reverse?: boolean;
  title: string;
  visual: ReactNode;
}) {
  return (
    <section
      aria-labelledby={id}
      className={cn(
        shellClasses,
        "grid grid-cols-1 items-center gap-group pt-chapter md:grid-cols-2 md:gap-section [&>*]:min-w-0",
      )}
    >
      <div className={cn(reverse && "md:order-2")}>
        <h2
          className="font-display text-title font-medium tracking-tight text-ink"
          id={id}
        >
          {title}
        </h2>
        <div className="mt-2 grid gap-3 text-lede text-mute">{children}</div>
      </div>
      {visual}
    </section>
  );
}
