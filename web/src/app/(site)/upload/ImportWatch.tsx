"use client";

import { AlertCircle } from "lucide-react";
import type { RefObject } from "react";
import { Button } from "@/components/ui/button";
import type { ImportStage } from "@/lib/import-stage";

export function ImportWatch({
  onCheckAgain,
  stage,
}: {
  onCheckAgain: () => void;
  stage: Extract<ImportStage, { at: "lost" | "reading" }>;
}) {
  const lost = stage.at === "lost";

  return (
    <section className="mt-10" aria-live="polite">
      <h2 className="font-display text-section font-medium text-ink">
        {lost ? "Illarin lost sight of your file" : stage.heading}
      </h2>
      <p className="mt-2 text-ui text-mute">
        {lost
          ? stage.message
          : "You can leave this page. Illarin carries on reading it, and this page keeps checking while it stays open."}
      </p>

      {lost ? (
        <Button className="mt-5" onClick={onCheckAgain} variant="primary">
          Check again
        </Button>
      ) : (
        <div className="mt-6 h-1 overflow-hidden rounded-control bg-deep">
          <div className="h-full w-2/5 rounded-control bg-action motion-safe:animate-sweep motion-reduce:w-full motion-reduce:opacity-45" />
        </div>
      )}
    </section>
  );
}

export function ImportRefusal({
  headingRef,
  message,
  onBeginAgain,
}: {
  headingRef: RefObject<HTMLHeadingElement | null>;
  message: string;
  onBeginAgain: () => void;
}) {
  return (
    <section className="mt-10">
      <h2
        className="flex items-center gap-2.5 font-display text-section font-medium text-ink"
        ref={headingRef}
        tabIndex={-1}
      >
        <AlertCircle
          aria-hidden="true"
          className="text-stop"
          size={22}
          strokeWidth={1.6}
        />
        This file was not added
      </h2>
      <p className="mt-2 text-ui text-mute">{message}</p>
      <Button className="mt-5" onClick={onBeginAgain} variant="primary">
        Choose another file
      </Button>
    </section>
  );
}
