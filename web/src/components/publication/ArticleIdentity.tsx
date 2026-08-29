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
            <span className={styles.release}>
              {release.app.name} {release.version}
            </span>
          ) : null}
          {publishedAt ? (
            <time dateTime={publishedAt}>{readableDate(publishedAt)}</time>
          ) : null}
          {updatedAt ? (
            <span className={styles.revised}>
              Updated {readableDate(updatedAt)}
            </span>
          ) : null}
        </p>
      </div>
    </header>
  );
}

// titleBand sizes the display type to the title rather than to the page.
function titleBand(title: string): "short" | "medium" | "long" {
  if (title.length > 78) return "long";
  if (title.length > 42) return "medium";
  return "short";
}

function Byline({ byline }: { byline: PostByline }) {
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
