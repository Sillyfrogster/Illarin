"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";
import { BLOG_HOME } from "@/lib/publication-metadata";

/** What the blog says when Illarin itself broke on the way to a post. */
export default function BlogError({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  return (
    <DeadEnd
      heading="The blog stopped short"
      line="Something here broke on our side. It is recorded, and trying again is often all it takes."
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
