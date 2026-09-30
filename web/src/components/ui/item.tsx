"use client";

import { Maximize2 } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";
import {
  useFluidHover,
  useRegisterFluidHoverItem,
} from "@/lib/use-fluid-hover";

const GroupContext = createContext<{
  register: (index: number, element: HTMLElement | null) => void;
  next: () => number;
} | null>(null);

/** ItemOpenContext lets each row open itself somewhere roomier than the list. */
export const ItemOpenContext = createContext<((key: string) => void) | null>(
  null,
);

/** ItemGroup is shadcn's item group on a quiet plate; the hover plate follows the pointer from row to row. */
export function ItemGroup({
  as: Tag = "ul",
  children,
  className,
  label,
}: {
  as?: "ul" | "ol" | "dl";
  children: ReactNode;
  className?: string;
  label?: string;
}) {
  const container = useRef<HTMLElement>(null);
  const hover = useFluidHover(container);
  const count = useRef(0);
  const next = useCallback(() => count.current++, []);
  const { registerItem } = hover;
  const group = useMemo(
    () => ({ register: registerItem, next }),
    [registerItem, next],
  );

  return (
    <GroupContext value={group}>
      <Tag
        aria-label={label}
        className={cn(
          "relative flex list-none flex-col overflow-hidden rounded-plate bg-inset",
          className,
        )}
        data-slot="item-group"
        onMouseEnter={hover.handlers.onMouseEnter}
        onMouseLeave={hover.handlers.onMouseLeave}
        onMouseMove={hover.handlers.onMouseMove}
        ref={(node: HTMLElement | null) => {
          container.current = node;
        }}
      >
        {children}
        <FluidHoverHighlight className="rounded-control" hover={hover} />
      </Tag>
    </GroupContext>
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
    <li className="relative z-10 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 bg-deep/70 px-4 py-2.5">
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
  const row = useRef<HTMLElement>(null);
  const group = useContext(GroupContext);
  const [index, setIndex] = useState<number>();
  useEffect(() => setIndex(group?.next()), [group]);
  useRegisterFluidHoverItem(group?.register, index, row);
  const open = useContext(ItemOpenContext);

  return (
    <Tag
      className={cn(
        "group/item relative z-10 flex min-w-0 flex-col gap-1.5 px-4 py-3.5 not-first:border-rule/45 not-first:border-t",
        open && itemKey && "pr-14",
        className,
      )}
      data-fluid-hover-index={index}
      data-slot="item"
      ref={(node: HTMLElement | null) => {
        row.current = node;
      }}
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
