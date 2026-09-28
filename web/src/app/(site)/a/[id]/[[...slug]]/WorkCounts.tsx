"use client";

import { type ReactNode, useEffect, useState } from "react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { api } from "@/lib/api/client";

/** Records this display as a work view and shows the lifetime counts, giving the creator the download split and followers outright */
export function WorkCounts({
  workId,
  isOwner,
  views,
  downloads,
  sends,
  followers,
}: {
  workId: string;
  isOwner: boolean;
  views: number;
  downloads: number;
  sends: number;
  followers?: number;
}) {
  useEffect(() => {
    if (!isOwner) api("POST", `/v1/works/${workId}/views`).catch(() => {});
  }, [workId, isOwner]);

  const split = [
    `${(downloads - sends).toLocaleString("en-US")} downloaded`,
    `${sends.toLocaleString("en-US")} sent to an app`,
  ];
  return (
    <dl className="mt-8 flex flex-wrap gap-x-10 gap-y-4">
      <Count label={views === 1 ? "View" : "Views"} value={views} />
      <Count
        label={downloads === 1 ? "Download" : "Downloads"}
        note={isOwner ? split : undefined}
        value={
          isOwner ? (
            downloads
          ) : (
            <DownloadSplit total={downloads} split={split} />
          )
        }
      />
      {followers === undefined ? null : (
        <Count
          label={followers === 1 ? "Follower" : "Followers"}
          value={followers}
        />
      )}
    </dl>
  );
}

function Count({
  label,
  note,
  value,
}: {
  label: string;
  note?: string[];
  value: number | ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="order-2 text-meta text-mute">{label}</dt>
      <dd className="order-1 font-display text-title font-medium tracking-tight text-ink tabular-nums">
        {typeof value === "number" ? value.toLocaleString("en-US") : value}
      </dd>
      {note ? (
        <dd className="order-3 text-label text-mute tabular-nums">
          {note.map((line) => (
            <span className="block" key={line}>
              {line}
            </span>
          ))}
        </dd>
      ) : null}
    </div>
  );
}

/** The reader's download count, which opens its split on hover, tap or keyboard focus and closes on Escape or a tap elsewhere */
function DownloadSplit({ total, split }: { total: number; split: string[] }) {
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [pinned, setPinned] = useState(false);
  return (
    <Popover
      open={hovered || focused || pinned}
      onOpenChange={(open) => {
        if (open) return;
        setHovered(false);
        setFocused(false);
        setPinned(false);
      }}
    >
      <PopoverTrigger asChild>
        <button
          className="relative rounded-control underline decoration-mute/50 decoration-dotted decoration-2 underline-offset-[0.2em] after:absolute after:inset-x-0 after:-inset-y-2 after:content-[''] hover:decoration-ink data-[state=open]:decoration-ink"
          onBlur={() => setFocused(false)}
          onClick={(event) => {
            event.preventDefault();
            setPinned(!pinned);
          }}
          onFocus={(event) =>
            setFocused(event.currentTarget.matches(":focus-visible"))
          }
          onPointerEnter={(event) => {
            if (event.pointerType === "mouse") setHovered(true);
          }}
          onPointerLeave={(event) => {
            if (event.pointerType === "mouse") setHovered(false);
          }}
          type="button"
        >
          {total.toLocaleString("en-US")}
          <span className="sr-only">, {split.join(", ")}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        aria-hidden="true"
        className="w-auto p-4 text-meta tabular-nums"
        onCloseAutoFocus={(event) => event.preventDefault()}
        onOpenAutoFocus={(event) => event.preventDefault()}
        sideOffset={32}
      >
        {split.map((line) => (
          <p key={line}>{line}</p>
        ))}
      </PopoverContent>
    </Popover>
  );
}
