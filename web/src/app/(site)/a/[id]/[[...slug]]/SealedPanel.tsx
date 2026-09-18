"use client";

import { Lock } from "lucide-react";
import { Button } from "@/components/ui/button";

export function SealedPanel({
  workId,
  count,
}: {
  workId: string;
  count: number;
}) {
  return (
    <section
      aria-labelledby="sealed-heading"
      className="rounded-plate bg-deep p-4"
    >
      <h2 className="text-ui font-medium text-ink" id="sealed-heading">
        Your sealed content
      </h2>
      <p className="mt-2 text-meta text-mute">
        {count === 1
          ? "One block of this preset was sealed on LumiHub"
          : `${count} blocks of this preset were sealed on LumiHub`}
        . Readers never saw them and still do not. Illarin has no way to put
        them back into a download, so take a copy and keep it.
      </p>
      <Button asChild className="mt-3 w-full">
        <a href={`/api/v1/works/${workId}/sealed`}>
          <Lock aria-hidden="true" />
          Download the sealed blocks
        </a>
      </Button>
    </section>
  );
}
