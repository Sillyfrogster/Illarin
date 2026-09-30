import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import { fontWeights } from "@/lib/font-weight";

/** WeightLabel turns semibold when chosen without moving its neighbours: an invisible semibold copy holds the width. */
export function WeightLabel({
  children,
  chosen,
  className,
  id,
}: {
  children: ReactNode;
  chosen: boolean;
  className?: string;
  id?: string;
}) {
  return (
    <span className={cn("inline-grid", className)} id={id}>
      <span
        aria-hidden="true"
        className="invisible col-start-1 row-start-1"
        style={{ fontVariationSettings: fontWeights.semibold }}
      >
        {children}
      </span>
      <span
        className="col-start-1 row-start-1 transition-[font-variation-settings] duration-80"
        style={{
          fontVariationSettings: chosen
            ? fontWeights.semibold
            : fontWeights.normal,
        }}
      >
        {children}
      </span>
    </span>
  );
}
