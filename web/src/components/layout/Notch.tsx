import type { ReactNode } from "react";

const RAIL = 56;
const DROP = 84;
const CORNER = 28;
const CONTROL = CORNER / 2;

const FILL = {
  left: `path('M0 0 H${CORNER} V${DROP} C${CONTROL} ${DROP} ${CONTROL} ${RAIL} 0 ${RAIL} Z')`,
  right: `path('M0 0 H${CORNER} V${RAIL} C${CONTROL} ${RAIL} ${CONTROL} ${DROP} 0 ${DROP} Z')`,
} as const;

const STROKE = {
  left: `M0 ${RAIL - 0.5} C${CONTROL} ${RAIL - 0.5} ${CONTROL} ${DROP - 0.5} ${CORNER} ${DROP - 0.5}`,
  right: `M0 ${DROP - 0.5} C${CONTROL} ${DROP - 0.5} ${CONTROL} ${RAIL - 0.5} ${CORNER} ${RAIL - 0.5}`,
} as const;

/** The curve that carries the header's edge down around the wordmark */
function Corner({ side }: { side: "left" | "right" }) {
  return (
    <div className="relative h-[var(--header-height)] w-7 shrink-0">
      <div
        className="absolute inset-0 bg-plane"
        style={{ clipPath: FILL[side] }}
      />
      <svg
        aria-hidden="true"
        viewBox={`0 0 ${CORNER} ${DROP}`}
        className="absolute inset-0 size-full"
        fill="none"
      >
        <path d={STROKE[side]} stroke="var(--v-edge)" strokeWidth="1" />
      </svg>
    </div>
  );
}

/** Two rails, two curves, and the bay the mark sits in */
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
    <div className="flex h-[var(--header-height)] items-start bg-field">
      <div className="flex h-14 min-w-0 flex-1 items-center gap-1 bg-plane pl-[var(--rail-inset)] shadow-[inset_0_-1px_0_var(--v-edge)]">
        {start}
      </div>
      <Corner side="left" />
      <div className="flex h-[var(--header-height)] shrink-0 items-center justify-center bg-plane px-5 pb-1 shadow-[inset_0_-1px_0_var(--v-edge)]">
        {centre}
      </div>
      <Corner side="right" />
      <div className="flex h-14 min-w-0 flex-1 items-center justify-end gap-2 bg-plane pr-[var(--rail-inset)] shadow-[inset_0_-1px_0_var(--v-edge)]">
        {end}
      </div>
    </div>
  );
}
