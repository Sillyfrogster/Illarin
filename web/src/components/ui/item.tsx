"use client";

import { Maximize2 } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useContext,
} from "react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";

/** ItemOpenContext lets each row open itself somewhere roomier than the list. */
export const ItemOpenContext = createContext<((key: string) => void) | null>(
  null,
);

/** ItemGroup is a run of rows on a quiet fill, ruled between them. */
export function ItemGroup({
  as: Tag = "ul",
  children,
  className,
  label,
}: {
  as?: "ul" | "ol";
  children: ReactNode;
  className?: string;
  label?: string;
}) {
  return (
    <Tag
      aria-label={label}
      className={cn(
        "flex list-none flex-col overflow-hidden rounded-art bg-inset",
        className,
      )}
      data-slot="item-group"
    >
      {children}
    </Tag>
  );
}

/** ItemGroupHeading names the rows below it inside a group. */
export function ItemGroupHeading({
  children,
  count,
}: {
  children: ReactNode;
  count?: string;
}) {
  return (
    <li className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 bg-deep/70 px-4 py-2.5">
      <span className="font-ui text-meta font-medium text-ink">{children}</span>
      {count ? (
        <span className="font-ui text-label text-mute tabular-nums">
          {count}
        </span>
      ) : null}
    </li>
  );
}

/** Item is one row of a group; given an itemKey inside ItemOpenContext, it offers to open itself. */
export function Item({
  as: Tag = "li",
  children,
  className,
  itemKey,
}: {
  as?: "li" | "div";
  children: ReactNode;
  className?: string;
  itemKey?: string;
}) {
  const open = useContext(ItemOpenContext);

  return (
    <Tag
      className={cn(
        "group/item relative flex min-w-0 flex-col gap-1.5 px-4 py-3.5 not-first:border-rule/45 not-first:border-t",
        open && itemKey && "pr-14",
        className,
      )}
      data-slot="item"
    >
      {children}
      {open && itemKey ? (
        <Tooltip content="Open item details">
          <Button
            aria-label="Open item details"
            className="absolute top-2.5 right-2.5 opacity-0 transition-opacity duration-80 group-hover/item:opacity-100 focus-visible:opacity-100 pointer-coarse:opacity-100"
            onClick={() => open(itemKey)}
            size="icon"
            variant="ghost"
          >
            <Maximize2 aria-hidden="true" />
          </Button>
        </Tooltip>
      ) : null}
    </Tag>
  );
}

export function ItemTitle({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      className={cn("font-ui text-ui font-medium text-ink", className)}
      data-slot="item-title"
      {...props}
    />
  );
}

export function ItemDescription({ className, ...props }: ComponentProps<"p">) {
  return (
    <p
      className={cn("font-ui text-meta text-mute", className)}
      data-slot="item-description"
      {...props}
    />
  );
}
