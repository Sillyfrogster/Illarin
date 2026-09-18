"use client";

import { BellPlus, BellRing } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { offersWatch } from "@/lib/asset-watch";
import { useAssetWatching } from "./state";

/** Asks, after a download or send, whether to hear when the asset updates. */
export function WatchOffer() {
  const watching = useAssetWatching();
  const [asked, setAsked] = useState(false);
  if (!watching) return null;

  if (asked && watching.watch.state === "following") {
    return (
      <output className="mt-4 flex items-start gap-2 text-meta text-mute">
        <BellRing
          aria-hidden="true"
          className="mt-0.5 size-3.5 shrink-0 text-accent"
        />
        Watching. You get a notification when this {watching.kind} updates.
      </output>
    );
  }
  if (!offersWatch(watching.watch, watching.notNow)) return null;

  return (
    <div className="mt-4">
      <p className="flex items-start gap-2 text-meta text-ink">
        <BellPlus
          aria-hidden="true"
          className="mt-0.5 size-3.5 shrink-0 text-accent"
        />
        Get a notification when this {watching.kind} updates?
      </p>
      <div className="mt-2 flex flex-wrap gap-2 pl-5.5">
        <Button
          loading={watching.pending}
          onClick={() => {
            setAsked(true);
            watching.change(true);
          }}
          size="compact"
        >
          Watch
        </Button>
        <Button
          disabled={watching.pending}
          onClick={watching.sayNotNow}
          size="compact"
          variant="ghost"
        >
          Not now
        </Button>
      </div>
      {asked && watching.failure ? (
        <p className="mt-2 text-meta text-stop" role="alert">
          {watching.failure}
        </p>
      ) : null}
    </div>
  );
}
