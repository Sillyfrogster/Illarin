"use client";

import { useEffect, useState } from "react";
import {
  readAssetUpdateAnnouncements,
  type WorkUpdateAnnouncement,
} from "@/lib/api/asset-destinations";
import { cn } from "@/lib/cn";
import { deliveryStanding, deliveryState } from "@/lib/delivery-standing";

export function AnnouncementStatus({ assetId }: { assetId: string }) {
  const [sent, setSent] = useState<WorkUpdateAnnouncement[]>([]);

  useEffect(() => {
    const controller = new AbortController();
    void readAssetUpdateAnnouncements(assetId, controller.signal).then(
      (answer) => {
        if (answer.value) setSent(answer.value.announcements);
      },
    );
    return () => controller.abort();
  }, [assetId]);

  if (sent.length === 0) return null;

  return (
    <section
      aria-labelledby="announced"
      className="mt-7 flex flex-col gap-3 border-t border-rule pt-5"
    >
      <h3 className="font-display text-ui font-medium text-ink" id="announced">
        Announced
      </h3>
      <ul className="flex list-none flex-col gap-1.5">
        {sent.map((one) => (
          <li
            className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 rounded-control bg-deep px-3 py-2 text-meta"
            key={one.id}
          >
            <span className="min-w-0 text-ink wrap-anywhere">
              {one.destination}
            </span>
            <span className="text-mute">Update {one.updateNumber}</span>
            <span className={cn(troubled(one) ? "text-stop" : "text-mute")}>
              {deliveryStanding(one)}
            </span>
          </li>
        ))}
      </ul>
      <p className="text-meta text-mute">
        Sending never changes what readers have. A failure leaves this update
        published.
      </p>
    </section>
  );
}

function troubled(one: WorkUpdateAnnouncement): boolean {
  const standing = deliveryState(one);
  return standing === "gaveUp" || standing === "stopped";
}
