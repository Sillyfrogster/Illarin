"use client";

import { Button } from "@/components/ui/button";
import type { WorkBlock } from "@/lib/api/query";
import { blockAudience } from "@/lib/work-page-content";
import { useWorkspace } from "./state";

const NOTE = "-mt-1 mb-5 font-ui text-label text-mute";

const PANEL =
  "-mt-1 mb-5 flex flex-col items-stretch justify-between gap-3 rounded-control bg-plane p-3 text-meta text-mute sm:flex-row sm:items-center";

/** Tells a creator when a reader does not meet this block where it sits */
export function BlockAudience({ block }: { block: WorkBlock }) {
  const workspace = useWorkspace();
  const audience = blockAudience(block);

  if (audience === "shown") return null;

  if (audience === "hidden") {
    return (
      <div className={PANEL}>
        <span>Hidden from readers. Downloads still include it.</span>
        <Button
          className="shrink-0"
          onClick={() => workspace.arrangement.setHidden(block.id, false)}
          size="compact"
        >
          Show block
        </Button>
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

  return <p className={NOTE}>Empty. Readers see it once you write in it.</p>;
}
