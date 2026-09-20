import { ArrowRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import type { PostSummary } from "@/lib/api/query";
import { archivePath, postPath } from "@/lib/blog-paths";
import { readableDate } from "@/lib/dates";
import type { PostCover } from "@/lib/post-cover";
import { Byline, BylineText } from "./Byline";
import { titleBand } from "./post-title";

const LEAD_TITLE = {
  short: "max-w-[18ch] text-[clamp(2rem,3.6vw,3.2rem)]",
  medium: "max-w-[22ch] text-[clamp(1.9rem,3vw,2.7rem)]",
  long: "max-w-[28ch] text-[clamp(1.7rem,2.4vw,2.2rem)]",
} as const;

export function ArchiveLead({
  cover,
  post,
}: {
  cover: PostCover | null;
  post: PostSummary;
}) {
  const address = postPath(post.slug);
  return (
    <article className="grid items-center gap-x-12 gap-y-7 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
      {cover ? (
        <Link
          className="group relative block overflow-hidden rounded-plate bg-deep"
          href={address}
          tabIndex={-1}
        >
          <Image
            alt={cover.alt}
            className="h-auto max-h-[26rem] w-full object-contain transition-transform duration-500 group-hover:scale-[1.02] motion-reduce:transform-none"
            height={cover.height}
            priority
            src={cover.url}
            unoptimized
            width={cover.width}
          />
          <span
            aria-hidden="true"
            className="absolute right-4 bottom-4 flex size-12 items-center justify-center rounded-full bg-field text-ink"
          >
            <ArrowRight className="size-5 -rotate-45 transition-transform duration-300 group-hover:rotate-0 motion-reduce:transition-none" />
          </span>
        </Link>
      ) : null}
      <div className={cover ? "min-w-0" : "min-w-0 xl:col-span-2"}>
        <h2
          className={`font-display font-medium tracking-[-0.025em] text-ink break-words text-balance ${LEAD_TITLE[titleBand(post.title)]} leading-[1.12]`}
        >
          <Link className="text-ink hover:text-accent" href={address}>
            {post.title}
          </Link>
        </h2>
        <p className="mt-4 flex flex-wrap items-baseline gap-x-3 text-meta text-mute">
          <Link
            className="font-medium text-accent hover:text-ink"
            href={archivePath("category", post.category.slug)}
          >
            {post.category.label}
          </Link>
          <When post={post} />
        </p>
        {post.summary ? (
          <p className="mt-5 max-w-[46ch] font-prose text-ui leading-7 text-mute">
            {post.summary}
          </p>
        ) : null}
        <div className="mt-7">
          <Byline byline={post.byline} />
        </div>
        <Link
          className="mt-6 inline-flex min-h-11 items-center gap-3 text-ui font-medium text-accent hover:text-ink"
          href={address}
        >
          Read post
          <ArrowRight aria-hidden="true" className="size-4" />
        </Link>
      </div>
    </article>
  );
}

export function ArchiveRow({
  narrowed,
  post,
}: {
  narrowed: "category" | null;
  post: PostSummary;
}) {
  return (
    <Link
      className="relative grid items-start gap-x-8 gap-y-3 py-6 sm:grid-cols-[8.5rem_minmax(0,1fr)_1.25rem]"
      href={postPath(post.slug)}
    >
      <span className="text-meta text-mute">
        <time dateTime={post.publishedAt}>
          {readableDate(post.publishedAt)}
        </time>
        {narrowed === "category" ? null : (
          <span className="mt-1 block text-accent">{post.category.label}</span>
        )}
      </span>
      <div className="min-w-0">
        <h3 className="max-w-[44ch] font-display text-section leading-snug font-medium text-ink text-balance">
          {post.title}
        </h3>
        {post.summary ? (
          <p className="mt-2 max-w-[64ch] font-prose text-ui leading-6 text-mute">
            {post.summary}
          </p>
        ) : null}
        <p className="mt-3 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-meta text-mute">
          <BylineText byline={post.byline} />
          {post.updatedAt ? (
            <time dateTime={post.updatedAt}>
              Updated {readableDate(post.updatedAt)}
            </time>
          ) : null}
        </p>
      </div>
      <ArrowRight
        aria-hidden="true"
        className="mt-1 hidden size-5 self-center text-mute sm:block"
      />
    </Link>
  );
}

function When({ post }: { post: PostSummary }) {
  return (
    <>
      <time dateTime={post.publishedAt}>{readableDate(post.publishedAt)}</time>
      {post.updatedAt ? (
        <time dateTime={post.updatedAt}>
          Updated {readableDate(post.updatedAt)}
        </time>
      ) : null}
    </>
  );
}
