"use client";

import { BellOff, BellPlus, BellRing } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { watchWords } from "@/lib/asset-watch";
import { cn } from "@/lib/cn";
import { useAssetWatching } from "./state";

/** The small bell beside the download button that shows and changes the reader's watch. */
export function WatchControl() {
  const watching = useAssetWatching();
  if (!watching) return null;
  const words = watchWords(watching.watch, watching.kind);
  const Icon =
    watching.watch.state === "stopped"
      ? BellOff
      : words.watching
        ? BellRing
        : BellPlus;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          aria-label={words.name}
          className={cn(
            "data-[state=open]:bg-deep",
            words.watching &&
              "bg-accent-wash text-accent hover:bg-accent-wash/70 hover:text-accent",
          )}
          size="icon"
          title={words.detail}
          variant="ghost"
        >
          <Icon aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        aria-label={`Updates to this ${watching.kind}`}
        className="w-[min(19rem,calc(100vw-2rem))]"
      >
        <p className="font-medium text-ink">
          {words.watching ? "Watching" : "Not watching"}
        </p>
        <p className="mt-1 text-meta text-mute">{words.detail}</p>
        <Button
          className="mt-4 w-full"
          loading={watching.pending}
          onClick={() => watching.change(!words.watching)}
          variant={words.watching ? "secondary" : "primary"}
        >
          {words.action}
        </Button>
        {watching.failure ? (
          <p className="mt-3 text-meta text-stop" role="alert">
            {watching.failure}
          </p>
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
