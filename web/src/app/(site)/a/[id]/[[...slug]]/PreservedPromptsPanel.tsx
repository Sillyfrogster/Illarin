"use client";

import { Lock } from "lucide-react";
import { Button } from "@/components/ui/button";

export function PreservedPromptsPanel({
  workId,
  count,
}: {
  workId: string;
  count: number;
}) {
  return (
    <section
      aria-labelledby="preserved-prompts-heading"
      className="rounded-plate bg-deep p-4"
    >
      <h2
        className="text-ui font-medium text-ink"
        id="preserved-prompts-heading"
      >
        Your preserved prompts
      </h2>
      <p className="mt-2 text-meta text-mute">
        {count === 1
          ? "One prompt of this preset was kept private on LumiHub"
          : `${count} prompts of this preset were kept private on LumiHub`}
        . Readers never saw them and still do not. Illarin has no way to put
        them back into a download, so take a copy and keep it.
      </p>
      <Button asChild className="mt-3 w-full">
        <a href={`/api/v1/works/${workId}/preserved-prompts`}>
          <Lock aria-hidden="true" />
          Download the preserved prompts
        </a>
      </Button>
    </section>
  );
}
