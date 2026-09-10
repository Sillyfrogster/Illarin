"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";

export default function SiteError({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  return (
    <DeadEnd
      heading="Illarin stopped short"
      line="Something here broke on our side. It is recorded, and trying again is often all it takes."
      note={error.digest ? `Reference ${error.digest}` : undefined}
    >
      <Button onClick={retry} variant="primary">
        Try again
      </Button>
      <Button asChild variant="ghost">
        <Link href="/browse">Browse the catalog</Link>
      </Button>
    </DeadEnd>
  );
}
