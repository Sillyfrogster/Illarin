import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";
import { BLOG_HOME } from "@/lib/publication-metadata";

/** What the blog says for an address it holds no post at. */
export default function BlogNotFound() {
  return (
    <DeadEnd
      heading="No post is here"
      line="The address may be wrong, or the post may never have been published."
    >
      <Button asChild variant="primary">
        <Link href={BLOG_HOME}>All posts</Link>
      </Button>
    </DeadEnd>
  );
}
