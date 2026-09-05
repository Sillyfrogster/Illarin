import Link from "next/link";
import type { PostByline, PostMedia, PostRelease } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./Article.module.css";
import { ArticleHeader, type Header } from "./ArticleHeader";
import { Byline } from "./Byline";
import { titleBand } from "./post-title";

export function ArticleIdentity({
  category,
  title,
  summary,
  release,
  byline,
  header,
  media,
  publishedAt,
  updatedAt,
  standing,
}: {
  category: string;
  title: string;
  summary: string;
  release: PostRelease | null;
  byline: PostByline | null;
  header?: Header | null;
  media?: PostMedia[];
  publishedAt: string | null;
  updatedAt: string | null;
  standing?: string;
}) {
  return (
    <header className={styles.identity}>
      <h1 className={styles.title} data-length={titleBand(title)}>
        {title}
      </h1>
      {summary ? <p className={styles.summary}>{summary}</p> : null}
      <div className={styles.band}>
        {byline ? (
          <Byline byline={byline} />
        ) : (
          <p className={styles.draft}>{standing}</p>
        )}
        <p className={styles.filed}>
          <span className={styles.category}>{category}</span>
          {release ? (
            <Link
              className={styles.release}
              href={`/blog/app/${release.app.slug}`}
            >
              {release.app.name} {release.version}
            </Link>
          ) : null}
          {publishedAt ? (
            <time dateTime={publishedAt}>{readableDate(publishedAt)}</time>
          ) : null}
          {updatedAt ? (
            <time className={styles.revised} dateTime={updatedAt}>
              Updated {readableDate(updatedAt)}
            </time>
          ) : null}
        </p>
      </div>
      <ArticleHeader header={header} media={media ?? []} />
    </header>
  );
}
