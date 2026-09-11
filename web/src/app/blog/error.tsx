"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";
import { BLOG_HOME } from "@/lib/publication-metadata";

export default function BlogError({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  return (
    <DeadEnd
      heading="The blog could not load"
      line="Illarin could not load this page. Try again."
      note={error.digest ? `Reference ${error.digest}` : undefined}
    >
      <Button onClick={retry} variant="primary">
        Try again
      </Button>
      <Button asChild variant="ghost">
        <Link href={BLOG_HOME}>All posts</Link>
      </Button>
    </DeadEnd>
  );
}
