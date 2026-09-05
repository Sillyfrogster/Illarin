import Link from "next/link";
import type { PostSummary } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./ArchiveEntry.module.css";
import { Byline, BylineLine } from "./Byline";

/** The newest post, set into the publication's masthead art. */
export function ArchiveLead({ post }: { post: PostSummary }) {
  return (
    <article className={styles.lead}>
      <h1 className={styles.leadTitle} data-length={titleBand(post.title)}>
        <Link href={`/blog/${post.slug}`}>{post.title}</Link>
      </h1>
      {post.summary ? (
        <p className={styles.leadSummary}>{post.summary}</p>
      ) : null}
      <div className={styles.leadBand}>
        <Byline byline={post.byline} />
        <p className={styles.leadFiled}>
          <Category post={post} />
          <App post={post} />
          <When post={post} />
        </p>
      </div>
    </article>
  );
}

/** One post in the ruled chronological list, without the fact its archive already states. */
export function ArchiveRow({
  post,
  narrowed,
}: {
  post: PostSummary;
  narrowed?: "category" | "app" | null;
}) {
  return (
    <article className={styles.row}>
      <h2 className={styles.rowTitle}>
        <Link className={styles.reach} href={`/blog/${post.slug}`}>
          {post.title}
        </Link>
      </h2>
      {post.summary ? (
        <p className={styles.rowSummary}>{post.summary}</p>
      ) : null}
      <p className={styles.rowFiled}>
        {narrowed === "category" ? null : <Category post={post} />}
        <BylineLine byline={post.byline} quiet={narrowed === "app"} />
        {narrowed === "app" ? null : <App post={post} />}
        <When post={post} />
      </p>
    </article>
  );
}

function Category({ post }: { post: PostSummary }) {
  return (
    <Link
      className={styles.category}
      href={`/blog/category/${post.category.slug}`}
    >
      {post.category.label}
    </Link>
  );
}

function App({ post }: { post: PostSummary }) {
  if (!post.app || !post.releaseVersion) return null;
  return (
    <Link className={styles.app} href={`/blog/app/${post.app.slug}`}>
      {post.app.name} {post.releaseVersion}
    </Link>
  );
}

function When({ post }: { post: PostSummary }) {
  return (
    <>
      <time dateTime={post.publishedAt}>{readableDate(post.publishedAt)}</time>
      {post.updatedAt ? (
        <time className={styles.revised} dateTime={post.updatedAt}>
          Updated {readableDate(post.updatedAt)}
        </time>
      ) : null}
    </>
  );
}

/** Sizes the lead's display type to the title rather than to the page. */
function titleBand(title: string): "short" | "medium" | "long" {
  if (title.length > 78) return "long";
  if (title.length > 42) return "medium";
  return "short";
}
