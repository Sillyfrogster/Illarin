"use client";

import { EyeOff, Trash2 } from "lucide-react";
import type { AssetBlock } from "@/lib/api/query";
import { LayoutPicker, WidthPicker } from "./ArrangementPickers";
import { useWorkspace } from "./workspace/state";

const TOOL =
  "inline-flex size-9 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-plane hover:text-ink";

/** The controls a block carries, held together so they read as one instrument. */
export function BlockTools({
  block,
  onHide,
  onIssue,
  onRemove,
  suggestedWidth,
}: {
  block: AssetBlock;
  onHide: () => void;
  onIssue: (message: string) => void;
  onRemove: () => void;
  suggestedWidth?: AssetBlock["width"];
}) {
  const workspace = useWorkspace();

  function change(changes: Partial<AssetBlock>) {
    workspace.setBlocks(
      workspace.blocks.map((item) =>
        item.id === block.id ? { ...item, ...changes } : item,
      ),
    );
  }

  return (
    <div
      className="flex shrink-0 items-center gap-0.5 rounded-plate bg-deep p-0.5 opacity-0 transition-opacity duration-200 group-focus-within/block:opacity-100 group-hover/block:opacity-100 motion-reduce:transition-none max-md:opacity-100"
      data-measurement-ignore
      role="toolbar"
      aria-label={`${block.title} controls`}
    >
      <WidthPicker
        inline
        layout={block.layout}
        onIssue={onIssue}
        onSelect={(width) => change({ width })}
        suggestedWidth={suggestedWidth}
        width={block.width}
      />
      {block.allowedLayouts.length > 1 ? (
        <LayoutPicker
          allowedLayouts={block.allowedLayouts}
          elementLabels={block.elements.map(
            (element) => element.label || "Content",
          )}
          inline
          layout={block.layout}
          onIssue={onIssue}
          onSelect={(layout) => change({ layout })}
          width={block.width}
        />
      ) : null}
      {block.hideable && !block.hidden ? (
        <button
          aria-label={`Hide ${block.title} from the public page`}
          className={TOOL}
          onClick={onHide}
          type="button"
        >
          <EyeOff aria-hidden="true" size={15} />
        </button>
      ) : null}
      {block.required ? null : (
        <button
          aria-label={`Remove ${block.title}`}
          className={TOOL}
          onClick={onRemove}
          type="button"
        >
          <Trash2 aria-hidden="true" size={15} />
        </button>
      )}
    </div>
  );
}
