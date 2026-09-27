"use client";

import { useEffect } from "react";
import { api } from "@/lib/api/client";

/** Records this display as a work view and shows the lifetime view and download counts */
export function WorkCounts({
  workId,
  isOwner,
  views,
  downloads,
}: {
  workId: string;
  isOwner: boolean;
  views: number;
  downloads: number;
}) {
  useEffect(() => {
    if (!isOwner) api("POST", `/v1/works/${workId}/views`).catch(() => {});
  }, [workId, isOwner]);

  return (
    <dl className="mt-8 flex gap-10">
      <Count value={views} one="View" many="Views" />
      <Count value={downloads} one="Download" many="Downloads" />
    </dl>
  );
}

function Count({
  value,
  one,
  many,
}: {
  value: number;
  one: string;
  many: string;
}) {
  return (
    <div className="flex flex-col-reverse gap-1">
      <dt className="text-meta text-mute">{value === 1 ? one : many}</dt>
      <dd className="font-display text-title font-medium tracking-tight text-ink tabular-nums">
        {value.toLocaleString("en-US")}
      </dd>
    </div>
  );
}
