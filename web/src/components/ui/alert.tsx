import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const alertVariants = cva(
  "block w-full rounded-control px-4 py-3 font-ui text-ui [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      tone: {
        stop: "bg-stop-wash text-stop",
        done: "bg-accent-wash text-accent",
        quiet: "bg-deep text-ink",
      },
    },
    defaultVariants: { tone: "quiet" },
  },
);

/** Alert is a notice on a page: a failure is announced at once, a finished step politely. */
export function Alert({
  className,
  tone,
  ...props
}: ComponentProps<"div"> & VariantProps<typeof alertVariants>) {
  return (
    <div
      aria-live={tone === "done" ? "polite" : undefined}
      className={cn(alertVariants({ tone }), className)}
      role={tone === "stop" ? "alert" : tone === "done" ? "status" : undefined}
      {...props}
    />
  );
}
