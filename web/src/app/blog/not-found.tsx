import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";
import { BLOG_HOME } from "@/lib/blog-paths";

export default function BlogNotFound() {
  return (
    <DeadEnd
      heading="Post not found"
      line="The address may be wrong, or the post may never have been published."
    >
      <Button asChild variant="primary">
        <Link href={BLOG_HOME}>All posts</Link>
      </Button>
    </DeadEnd>
  );
}
