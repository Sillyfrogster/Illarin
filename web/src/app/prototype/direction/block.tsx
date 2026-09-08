"use client";

import type { Block, Layout, Width } from "./assets";
import { ElementView } from "./element";
import { Arrive } from "./motion";
import { cn, type Direction } from "./ui";

const SPAN: Record<Width, string> = {
  full: "vd:lg:col-span-12",
  two_thirds: "vd:lg:col-span-8",
  half: "vd:lg:col-span-6",
  third: "vd:lg:col-span-4",
};

/** Six arrangements of slots, and nothing a creator can measure themselves */
const SLOTS: Record<Layout, string> = {
  single: "vd:grid-cols-1",
  duo: "vd:grid-cols-1 vd:md:grid-cols-2",
  "main-aside": "vd:grid-cols-1 vd:md:grid-cols-[2fr_1fr]",
  trio: "vd:grid-cols-1 vd:md:grid-cols-3",
  "stack-2": "vd:grid-cols-1",
  "stack-3": "vd:grid-cols-1",
};

const GAP: Record<Layout, string> = {
  single: "",
  duo: "vd:gap-x-10 vd:gap-y-group",
  "main-aside": "vd:gap-x-10 vd:gap-y-group",
  trio: "vd:gap-x-10 vd:gap-y-group",
  "stack-2": "vd:gap-group",
  "stack-3": "vd:gap-group",
};

export function BlockView({
  block,
  direction,
  number,
}: {
  block: Block;
  direction: Direction;
  number?: number;
}) {
  const labelled = block.elements.length > 1 || block.layout !== "single";
  return (
    <Arrive className={cn("vd:min-w-0", SPAN[block.width])}>
      <BlockHead block={block} direction={direction} number={number} />
      <div
        className={cn(
          "vd:mt-6 vd:grid",
          SLOTS[block.layout],
          GAP[block.layout],
          direction === "ambient" && "vd:pl-5",
        )}
      >
        {block.elements.map((element) => (
          <ElementView
            key={element.id}
            element={element}
            direction={direction}
            labelled={labelled}
          />
        ))}
      </div>
    </Arrive>
  );
}

/* A heading is sized by the room its block was given, so a narrow block never shouts */
const HEAD_SIZE: Record<Width, string> = {
  full: "vd:text-title",
  two_thirds: "vd:text-title",
  half: "vd:text-section",
  third: "vd:text-section",
};

function BlockHead({
  block,
  direction,
  number,
}: {
  block: Block;
  direction: Direction;
  number?: number;
}) {
  const size = HEAD_SIZE[block.width];
  if (direction === "ledger") {
    return (
      <div className="vd:flex vd:items-baseline vd:gap-4 vd:shadow-[inset_0_-1px_0_var(--v-rule)] vd:pb-3">
        {number !== undefined && (
          <span
            className={cn("v-tabular vd:font-normal", size)}
            style={{ color: "var(--v-accent)" }}
          >
            {String(number).padStart(2, "0")}
          </span>
        )}
        <h2 className={cn("vd:font-display vd:font-medium", size)}>
          {block.title}
        </h2>
      </div>
    );
  }
  if (direction === "ambient") {
    return (
      <div className="vd:flex vd:items-baseline vd:gap-4">
        <span
          aria-hidden="true"
          className="vd:mt-1 vd:h-7 vd:w-1 vd:shrink-0 vd:rounded-full"
          style={{ background: "var(--v-tint-a)" }}
        />
        <h2 className={cn("vd:font-display vd:font-medium", size)}>
          {block.title}
        </h2>
      </div>
    );
  }
  return (
    <div>
      <h2 className={cn("vd:font-display vd:font-medium", size)}>
        {block.title}
      </h2>
      <div className="vd:mt-5 vd:h-px vd:w-full vd:bg-rule" />
    </div>
  );
}
