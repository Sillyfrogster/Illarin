"use client";

import { AlertCircle, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { ReadinessItem } from "@/lib/api/query";
import { type ReadinessTarget, readinessTarget } from "@/lib/readiness";

export function ReadinessList({
  items,
  onGo,
}: {
  items: ReadinessItem[];
  onGo: (target: ReadinessTarget) => void;
}) {
  return (
    <ul className="flex list-none flex-col gap-4">
      {items.map((item) => {
        const target = item.met ? null : readinessTarget(item);
        return (
          <li className="flex gap-3" key={item.id}>
            <span
              aria-hidden="true"
              className={item.met ? "mt-0.5 text-accent" : "mt-0.5 text-stop"}
            >
              {item.met ? <Check size={16} /> : <AlertCircle size={16} />}
            </span>
            <span className="min-w-0">
              <span className="block text-ui font-medium text-ink">
                {item.label}
              </span>
              <span className="mt-0.5 block text-meta text-mute">
                {item.detail}
              </span>
              {target ? (
                <Button
                  className="mt-1 min-h-control text-meta"
                  onClick={() => onGo(target)}
                  variant="link"
                >
                  Edit required content
                </Button>
              ) : null}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
