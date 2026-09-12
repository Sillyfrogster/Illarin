import type { ReactNode } from "react";

const RAIL = 72;
const DROP = 96;
const CORNER = 28;
const CONTROL = CORNER / 2;

const FILL = {
  left: `path('M0 0 H${CORNER} V${DROP} C${CONTROL} ${DROP} ${CONTROL} ${RAIL} 0 ${RAIL} Z')`,
  right: `path('M0 0 H${CORNER} V${RAIL} C${CONTROL} ${RAIL} ${CONTROL} ${DROP} 0 ${DROP} Z')`,
} as const;

function Corner({ side }: { side: "left" | "right" }) {
  return (
    <div className="relative h-[var(--header-bay-height)] w-7 shrink-0">
      <div
        className="absolute inset-0 bg-plane"
        style={{ clipPath: FILL[side] }}
      />
    </div>
  );
}

export function Notch({
  start,
  centre,
  end,
}: {
  start: ReactNode;
  centre: ReactNode;
  end: ReactNode;
}) {
  return (
    <div className="pointer-events-none flex h-[var(--header-bay-height)] items-start">
      <div className="pointer-events-auto flex h-[var(--header-height)] min-w-0 flex-1 items-center gap-1 bg-plane pl-[var(--rail-inset)]">
        {start}
      </div>
      <Corner side="left" />
      <div className="pointer-events-auto flex h-[var(--header-bay-height)] shrink-0 items-center justify-center bg-plane px-3 pb-3 sm:px-5">
        {centre}
      </div>
      <Corner side="right" />
      <div className="pointer-events-auto flex h-[var(--header-height)] min-w-0 flex-1 items-center justify-end gap-3 bg-plane pr-[var(--rail-inset)]">
        {end}
      </div>
    </div>
  );
}
