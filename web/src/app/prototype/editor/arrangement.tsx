"use client";

import { motion } from "framer-motion";
import { ArrowDown, ArrowUp, Eye, EyeOff } from "lucide-react";
import { useState } from "react";
import {
  BLOCK_WIDTHS,
  type BlockLayout,
  type BlockWidth,
  LAYOUT_LABELS,
  layoutChoiceIssue,
  NARROW_BLOCK_GRID_PX,
  packBlockRows,
  WIDTH_LABELS,
  widthChoiceIssue,
} from "@/lib/page-arrangement";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import { type Asset, type Block, blockRules } from "./data";
import { Button, cn, Field, Input, Notice, Select } from "./ui";

function isEmpty(block: Block) {
  return block.elements.every(
    (element) =>
      !element.text.trim() && !element.items.some((item) => item.text.trim()),
  );
}

/** The page shape in miniature, where clicking a block selects it */
function ShapeMap({
  asset,
  selected,
  select,
  width,
}: {
  asset: Asset;
  selected: string;
  select: (id: string) => void;
  width?: number;
}) {
  const rows = packBlockRows(asset.blocks, { availableWidth: width });
  return (
    <div className="ws:space-y-2 ws:rounded-2xl ws:bg-ink/5 ws:p-3">
      {rows.map((row) => (
        <div key={row[0].block.id} className="ws:grid ws:grid-cols-12 ws:gap-2">
          {row.map(({ block, columns, startColumn }) => {
            const quiet = block.hidden || isEmpty(block);
            return (
              <button
                key={block.id}
                type="button"
                onClick={() => select(block.id)}
                style={{ gridColumn: `${startColumn} / span ${columns}` }}
                aria-current={block.id === selected ? "true" : undefined}
                className={cn(
                  "ws:relative ws:min-h-14 ws:rounded-lg ws:px-3 ws:py-2.5 ws:text-left ws:text-xs ws:font-semibold ws:wrap-anywhere ws:transition ws:motion-reduce:transition-none",
                  quiet
                    ? "ws:bg-transparent ws:text-mute ws:shadow-[inset_0_0_0_1px_var(--w-line)]"
                    : "ws:bg-card ws:text-ink ws:shadow-[inset_0_0_0_1px_var(--w-line)]",
                  block.id === selected &&
                    "ws:shadow-[inset_0_0_0_2px_var(--w-ink)]",
                )}
              >
                {block.id === selected && (
                  <motion.span
                    layoutId="arrangement-mark"
                    aria-hidden="true"
                    className="ws:absolute ws:inset-x-2.5 ws:bottom-1.5 ws:h-[2px] ws:rounded-full"
                    style={{ background: "var(--w-spectrum)" }}
                  />
                )}
                {block.title || "Untitled block"}
                <span className="ws:mt-1 ws:block ws:font-normal ws:text-mute">
                  {block.hidden
                    ? "Hidden"
                    : isEmpty(block)
                      ? "Empty"
                      : WIDTH_LABELS[block.width]}
                </span>
              </button>
            );
          })}
        </div>
      ))}
    </div>
  );
}

export function Arrangement({
  asset,
  update,
  readOnly,
  selected,
  select,
}: {
  asset: Asset;
  update: (asset: Asset) => void;
  readOnly: boolean;
  selected: string;
  select: (id: string) => void;
}) {
  const [ref, width] = useMeasuredWidth<HTMLDivElement>();
  const [issue, setIssue] = useState<string>();
  const narrow = width !== undefined && width <= NARROW_BLOCK_GRID_PX;
  const index = asset.blocks.findIndex((block) => block.id === selected);
  const block = asset.blocks[index] ?? asset.blocks[0];
  const rules = blockRules[block.definition];

  function patchBlock(change: Partial<Block>) {
    update({
      ...asset,
      blocks: asset.blocks.map((b) =>
        b.id === block.id ? { ...b, ...change } : b,
      ),
    });
  }
  function move(delta: number) {
    const to = index + delta;
    if (to < 0 || to >= asset.blocks.length) return;
    const blocks = [...asset.blocks];
    [blocks[index], blocks[to]] = [blocks[to], blocks[index]];
    update({ ...asset, blocks });
  }

  return (
    <div ref={ref} className="ws:space-y-6">
      <ShapeMap
        asset={asset}
        selected={block.id}
        select={select}
        width={width}
      />
      {narrow && (
        <Notice>
          Blocks fill the width on a phone. Your desktop widths are kept.
        </Notice>
      )}
      {issue && <Notice tone="critical">{issue}</Notice>}
      <fieldset
        disabled={readOnly}
        className="ws:space-y-6 ws:rounded-2xl ws:bg-card ws:p-5 ws:shadow-[inset_0_0_0_1px_var(--w-line)] ws:md:p-6"
      >
        <legend className="ws:sr-only">Selected block</legend>
        <div className="ws:grid ws:gap-5 ws:md:grid-cols-2">
          <Field label="Block title">
            <Input
              value={block.title}
              onChange={(e) => patchBlock({ title: e.target.value })}
            />
          </Field>
          <div className="ws:grid ws:grid-cols-2 ws:gap-4">
            {!narrow && (
              <Field label="Width">
                <Select
                  aria-label={`${block.title} width`}
                  value={block.width}
                  onChange={(e) => {
                    const next = e.target.value as BlockWidth;
                    const reason = widthChoiceIssue(block.layout, next);
                    setIssue(reason ?? undefined);
                    if (!reason) patchBlock({ width: next });
                  }}
                >
                  {BLOCK_WIDTHS.map((value) => (
                    <option key={value} value={value}>
                      {WIDTH_LABELS[value]}
                    </option>
                  ))}
                </Select>
              </Field>
            )}
            <Field label="Layout">
              <Select
                aria-label={`${block.title} layout`}
                value={block.layout}
                onChange={(e) => {
                  const next = e.target.value as BlockLayout;
                  const reason = layoutChoiceIssue(
                    next,
                    block.width,
                    block.elements.map((element) => element.label),
                  );
                  setIssue(reason ?? undefined);
                  if (!reason) patchBlock({ layout: next });
                }}
              >
                {rules.layouts.map((layout) => (
                  <option key={layout} value={layout}>
                    {LAYOUT_LABELS[layout]}
                  </option>
                ))}
              </Select>
            </Field>
          </div>
        </div>
        <div className="ws:flex ws:flex-wrap ws:items-center ws:gap-2">
          <Button
            size="small"
            variant="outline"
            disabled={index <= 0}
            onClick={() => move(-1)}
          >
            <ArrowUp />
            Earlier
          </Button>
          <Button
            size="small"
            variant="outline"
            disabled={index === asset.blocks.length - 1}
            onClick={() => move(1)}
          >
            <ArrowDown />
            Later
          </Button>
          <span className="ws:text-xs ws:text-mute">
            Block {index + 1} of {asset.blocks.length}
          </span>
          {rules.hideable ? (
            <Button
              size="small"
              variant="outline"
              className="ws:ml-auto"
              onClick={() => patchBlock({ hidden: !block.hidden })}
            >
              {block.hidden ? <Eye /> : <EyeOff />}
              {block.hidden ? "Show on the page" : "Hide from the page"}
            </Button>
          ) : (
            <span className="ws:ml-auto ws:text-xs ws:text-mute">
              Required · always visible
            </span>
          )}
        </div>
        {block.hidden && (
          <p className="ws:text-sm ws:text-mute">
            Hidden on the page. Its content is kept and still travels in
            downloads.
          </p>
        )}
      </fieldset>
    </div>
  );
}
