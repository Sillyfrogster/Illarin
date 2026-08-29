import Image from "next/image";
import Link from "next/link";
import type { PostByline, PostRelease } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./Article.module.css";

export function ArticleIdentity({
  category,
  title,
  summary,
  release,
  byline,
  publishedAt,
  updatedAt,
  standing,
}: {
  category: string;
  title: string;
  summary: string;
  release: PostRelease | null;
  byline: PostByline | null;
  publishedAt: string | null;
  updatedAt: string | null;
  standing?: string;
}) {
  return (
    <header className={styles.identity}>
      <p className={styles.eyebrow}>
        <span className={styles.category}>{category}</span>
        {release ? (
          <span className={styles.release}>
            {release.app.name} {release.version}
          </span>
        ) : null}
      </p>
      <h1 className={styles.title}>{title}</h1>
      {summary ? <p className={styles.summary}>{summary}</p> : null}
      {byline ? (
        <Byline
          byline={byline}
          publishedAt={publishedAt}
          updatedAt={updatedAt}
        />
      ) : (
        <p className={styles.standing}>{standing}</p>
      )}
    </header>
  );
}

function Byline({
  byline,
  publishedAt,
  updatedAt,
}: {
  byline: PostByline;
  publishedAt: string | null;
  updatedAt: string | null;
}) {
  const name = byline.displayName || `@${byline.handle}`;
  return (
    <div className={styles.byline}>
      <span className={styles.portrait} aria-hidden="true">
        {byline.avatar ? (
          <Image
            alt=""
            height={44}
            src={byline.avatar.url}
            unoptimized
            width={44}
          />
        ) : (
          <span className={styles.monogram}>
            {byline.handle.slice(0, 1).toUpperCase()}
          </span>
        )}
      </span>
      <span className={styles.who}>
        <Link className={styles.name} href={`/@${byline.handle}`}>
          {name}
        </Link>
        <span className={styles.standing}>
          <Attribution byline={byline} />
        </span>
      </span>
      {publishedAt ? (
        <span className={styles.when}>
          <time dateTime={publishedAt}>{readableDate(publishedAt)}</time>
          {updatedAt ? (
            <span className={styles.revised}>
              Updated {readableDate(updatedAt)}
            </span>
          ) : null}
        </span>
      ) : null}
    </div>
  );
}

function Attribution({ byline }: { byline: PostByline }) {
  if (byline.app) {
    return (
      <>
        <span className={styles.affiliation}>{byline.app.name}</span>
        <span className={styles.publisher}>Published by Illarin</span>
      </>
    );
  }
  const standing = [...byline.positions, ...byline.distinctions];
  return (
    <>
      <span className={styles.affiliation}>Illarin Team</span>
      {standing.length > 0 ? (
        <span className={styles.publisher}>{standing.join(" · ")}</span>
      ) : null}
    </>
  );
}
