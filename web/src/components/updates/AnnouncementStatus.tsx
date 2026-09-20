"use client";

import { useEffect, useState } from "react";
import {
  readWorkAnnouncementAttempts,
  type WorkAnnouncementAttempt,
} from "@/lib/api/integrations";
import { attemptStanding, attemptState } from "@/lib/attempt-standing";
import { cn } from "@/lib/cn";

export function AnnouncementStatus({ workId }: { workId: string }) {
  const [sent, setSent] = useState<WorkAnnouncementAttempt[]>([]);

  useEffect(() => {
    const controller = new AbortController();
    void readWorkAnnouncementAttempts(workId, controller.signal).then(
      (answer) => {
        if (answer.value) setSent(answer.value.attempts);
      },
    );
    return () => controller.abort();
  }, [workId]);

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
              {one.integration}
            </span>
            <span className="text-mute">Update {one.versionNumber}</span>
            <span className={cn(troubled(one) ? "text-stop" : "text-mute")}>
              {attemptStanding(one)}
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

function troubled(one: WorkAnnouncementAttempt): boolean {
  const standing = attemptState(one);
  return standing === "gaveUp" || standing === "stopped";
}
