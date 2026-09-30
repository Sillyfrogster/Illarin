"use client";

import { cva, type VariantProps } from "class-variance-authority";
import Link from "next/link";
import { type ComponentProps, useId, useState } from "react";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/cn";

const badgeVariants = cva(
  "inline-flex max-w-full items-center gap-1.5 rounded-control font-ui font-medium [overflow-wrap:anywhere]",
  {
    variants: {
      tone: {
        quiet: "bg-deep text-ink",
        accent: "bg-accent-wash text-accent",
        stop: "bg-stop-wash text-stop",
      },
      size: {
        default: "min-h-6 px-2.5 py-0.5 text-label",
        compact: "min-h-5 px-2 text-[0.6875rem]",
      },
    },
    defaultVariants: { tone: "quiet", size: "default" },
  },
);

type BadgeProps = ComponentProps<"span"> & VariantProps<typeof badgeVariants>;

/** Badge is Fluid Functionalism's badge in the site's tones: a tag, a status or a small label. */
function Badge({ className, tone, size, ...props }: BadgeProps) {
  return (
    <span
      className={cn(badgeVariants({ tone, size }), className)}
      data-slot="badge"
      {...props}
    />
  );
}

/** BadgeLink is a badge that goes somewhere, washing violet under the pointer. */
function BadgeLink({ className, ...props }: ComponentProps<typeof Link>) {
  return (
    <Link
      className={cn(
        badgeVariants(),
        "transition-colors duration-80 hover:bg-accent-wash hover:text-accent motion-reduce:transition-none",
        className,
      )}
      {...props}
    />
  );
}

export type BadgeItem = {
  id: string;
  label: string;
  href?: string;
};

/** BadgeList wraps a run of badges, holding back all but the first few behind "N more" when given a limit. */
function BadgeList({
  items,
  limit,
  className,
}: {
  items: readonly BadgeItem[];
  limit?: number;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const id = useId();
  const held = limit === undefined ? 0 : Math.max(items.length - limit, 0);
  const shown = held > 0 && !open ? items.slice(0, limit) : items;

  return (
    <Collapsible className={className} onOpenChange={setOpen} open={open}>
      <ul className="flex list-none flex-wrap items-center gap-1.5" id={id}>
        {shown.map((item) => (
          <li className="max-w-full" key={item.id}>
            {item.href ? (
              <BadgeLink href={item.href}>{item.label}</BadgeLink>
            ) : (
              <Badge>{item.label}</Badge>
            )}
          </li>
        ))}
        {held > 0 ? (
          <li>
            <CollapsibleTrigger asChild>
              <Button aria-controls={id} size="compact" variant="ghost">
                {open ? "Fewer" : `${held} more`}
              </Button>
            </CollapsibleTrigger>
          </li>
        ) : null}
      </ul>
    </Collapsible>
  );
}

export { Badge, BadgeLink, BadgeList, badgeVariants };
