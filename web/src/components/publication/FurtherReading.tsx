import { ArrowRight } from "lucide-react";
import Link from "next/link";
import type { PostSummary } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";

export function FurtherReading({ posts }: { posts: PostSummary[] }) {
  if (posts.length === 0) return null;
  return (
    <section aria-labelledby="further-reading" className="mt-section">
      <h2
        className="mb-2 font-display text-section font-medium"
        id="further-reading"
      >
        Keep reading
      </h2>
      <ul className="list-none">
        {posts.map((post) => (
          <li key={post.id}>
            <Link
              className="group -mx-4 flex min-h-14 flex-wrap items-center justify-between gap-x-5 gap-y-1 rounded-plate px-4 py-3 transition-colors hover:bg-deep"
              href={`/blog/${post.slug}`}
            >
              <span className="min-w-0">
                <span className="block max-w-[52ch] font-display text-ui font-medium text-ink">
                  {post.title}
                </span>
                <span className="mt-1 flex flex-wrap items-baseline gap-x-3 text-meta text-mute">
                  <span className="text-accent">{post.category.label}</span>
                  {post.app ? <span>{post.app.name}</span> : null}
                  <time dateTime={post.publishedAt}>
                    {readableDate(post.publishedAt)}
                  </time>
                </span>
              </span>
              <ArrowRight
                aria-hidden="true"
                className="size-4 shrink-0 text-mute transition-transform duration-200 group-hover:translate-x-1 motion-reduce:transition-none"
              />
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
