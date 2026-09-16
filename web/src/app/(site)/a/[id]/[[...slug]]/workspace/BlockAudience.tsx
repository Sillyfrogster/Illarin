"use client";

import type { AssetBlock } from "@/lib/api/query";
import { blockAudience } from "@/lib/asset-page-content";
import { useWorkspace } from "./state";

const NOTE = "-mt-1 mb-5 font-ui text-label text-mute";

const PANEL =
  "-mt-1 mb-5 flex flex-col items-stretch justify-between gap-3 rounded-control bg-plane p-3 text-meta text-mute sm:flex-row sm:items-center";

const SHOW =
  "min-h-11 shrink-0 rounded-control bg-deep px-3 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45";

/** Tells a creator when a reader does not meet this block where it sits */
export function BlockAudience({ block }: { block: AssetBlock }) {
  const workspace = useWorkspace();
  const audience = blockAudience(block);

  if (audience === "shown") return null;

  if (audience === "hidden") {
    return (
      <div className={PANEL}>
        <span>
          Hidden from readers. This content is still included in downloads.
        </span>
        <button
          className={SHOW}
          onClick={() => workspace.arrangement.setHidden(block.id, false)}
          type="button"
        >
          Show block
        </button>
      </div>
    );
  }

  if (audience === "model") {
    return (
      <p className={NOTE}>
        Readers find this in the Model instructions panel at the foot of the
        page.
      </p>
    );
  }

  return (
    <p className={NOTE}>
      Empty. The block stays on your page, and readers see it as soon as you
      write in it.
    </p>
  );
}
