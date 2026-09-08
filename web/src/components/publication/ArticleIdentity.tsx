import Link from "next/link";
import type { PostByline, PostMedia, PostRelease } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import { ArticleHeader, type Header } from "./ArticleHeader";
import { Byline } from "./Byline";
import { titleBand } from "./post-title";

const TITLE = {
  short: "max-w-[18ch] text-hero",
  medium: "max-w-[22ch] text-[clamp(2.1rem,4.3vw,3.4rem)]",
  long: "max-w-[28ch] text-[clamp(1.9rem,3.2vw,2.6rem)]",
} as const;

/** Everything a post is, said once, above the writing itself. */
export function ArticleIdentity({
  byline,
  category,
  categorySlug,
  header,
  media,
  publishedAt,
  release,
  standing,
  summary,
  title,
  updatedAt,
}: {
  byline: PostByline | null;
  category: string;
  categorySlug?: string;
  header?: Header | null;
  media?: PostMedia[];
  publishedAt: string | null;
  release: PostRelease | null;
  standing?: string;
  summary: string;
  title: string;
  updatedAt: string | null;
}) {
  return (
    <header>
      <h1
        className={`font-display font-medium tracking-[-0.03em] break-words text-balance ${TITLE[titleBand(title)]}`}
      >
        {title}
      </h1>
      {summary ? (
        <p className="mt-6 max-w-[54ch] font-prose text-lede text-mute">
          {summary}
        </p>
      ) : null}
      <p className="mt-7 flex flex-wrap items-baseline gap-x-5 gap-y-2 text-meta text-mute">
        {categorySlug ? (
          <Link
            className="font-medium text-accent hover:text-ink"
            href={`/blog/category/${categorySlug}`}
          >
            {category}
          </Link>
        ) : (
          <span className="font-medium text-accent">{category}</span>
        )}
        {publishedAt ? (
          <time dateTime={publishedAt}>{readableDate(publishedAt)}</time>
        ) : null}
        {updatedAt ? (
          <time dateTime={updatedAt}>Updated {readableDate(updatedAt)}</time>
        ) : null}
        {release ? (
          <Link
            className="text-ink hover:text-accent"
            href={`/blog/app/${release.app.slug}`}
          >
            {release.app.name} {release.version}
          </Link>
        ) : null}
      </p>
      <div className="mt-6">
        {byline ? (
          <Byline byline={byline} />
        ) : (
          <p className="max-w-[34ch] text-meta text-mute">{standing}</p>
        )}
      </div>
      <ArticleHeader header={header} media={media ?? []} />
    </header>
  );
}
