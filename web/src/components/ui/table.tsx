"use client";

import {
  type ComponentProps,
  createContext,
  useContext,
  useMemo,
  useRef,
} from "react";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { cn } from "@/lib/cn";
import { fontWeights } from "@/lib/font-weight";
import { useSize } from "@/lib/size-context";
import {
  useFluidHover,
  useRegisterFluidHoverItem,
} from "@/lib/use-fluid-hover";

type TableContextValue = {
  registerItem: (index: number, element: HTMLElement | null) => void;
  activeIndex: number | null;
};

const TableContext = createContext<TableContextValue | null>(null);

/** Table is Fluid Functionalism's table: a hover plate follows the pointer down the body rows, which each need an index. */
function Table({ className, ...props }: ComponentProps<"table">) {
  const container = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(container);
  const { registerItem, activeIndex } = hover;
  const context = useMemo(
    () => ({ registerItem, activeIndex }),
    [registerItem, activeIndex],
  );

  return (
    <TableContext.Provider value={context}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: the pointer only moves the hover plate */}
      <div
        className="relative w-full overflow-x-auto"
        onMouseEnter={hover.handlers.onMouseEnter}
        onMouseLeave={hover.handlers.onMouseLeave}
        onMouseMove={hover.handlers.onMouseMove}
        ref={container}
      >
        <FluidHoverHighlight hover={hover} />
        <table
          className={cn("w-full border-collapse font-ui text-meta", className)}
          {...props}
        />
      </div>
    </TableContext.Provider>
  );
}

function TableHeader(props: ComponentProps<"thead">) {
  return <thead {...props} />;
}

function TableBody(props: ComponentProps<"tbody">) {
  return <tbody {...props} />;
}

/** TableRow joins the hover plate when given its index among the body rows; its rule hides beside the lit row. */
function TableRow({
  index,
  className,
  style,
  ...props
}: ComponentProps<"tr"> & { index?: number }) {
  const row = useRef<HTMLTableRowElement>(null);
  const context = useContext(TableContext);
  useRegisterFluidHoverItem(context?.registerItem, index, row);

  const isBodyRow = index !== undefined;
  const active = context?.activeIndex ?? null;
  const hideRule =
    active !== null &&
    ((isBodyRow && (index === active || index === active - 1)) ||
      (!isBodyRow && active === 0));

  return (
    <tr
      className={cn(
        "group/row relative z-10 border-b transition-[border-color] duration-80 last:border-b-0",
        hideRule ? "border-transparent" : "border-rule",
        isBodyRow && active === index && "is-active",
        className,
      )}
      data-fluid-hover-index={index}
      ref={row}
      style={{
        ...style,
        fontVariationSettings: isBodyRow
          ? fontWeights.normal
          : fontWeights.semibold,
      }}
      {...props}
    />
  );
}

function TableHead({ className, ...props }: ComponentProps<"th">) {
  const compact = useSize().variant === "compact";
  return (
    <th
      className={cn(
        "text-left align-middle whitespace-nowrap text-ink",
        compact ? "px-2.5 py-[5px]" : "px-3 py-2",
        className,
      )}
      {...props}
    />
  );
}

function TableCell({ className, ...props }: ComponentProps<"td">) {
  const compact = useSize().variant === "compact";
  return (
    <td
      className={cn(
        "align-middle whitespace-nowrap text-mute transition-colors duration-80 group-[.is-active]/row:text-ink",
        compact ? "px-2.5 py-[5px]" : "px-3 py-2",
        className,
      )}
      {...props}
    />
  );
}

export { Table, TableBody, TableCell, TableHead, TableHeader, TableRow };
