import Link from "next/link";
import type { PostSummary } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./ArchiveEntry.module.css";
import { Byline, BylineLine } from "./Byline";
import { titleBand } from "./post-title";

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
          <App app={filedApp(post)} version={post.releaseVersion} />
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
  const app = narrowed === "app" ? null : filedApp(post);
  const bylineRepeats =
    narrowed === "app" || (app !== null && app.slug === post.byline.app?.slug);
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
        <BylineLine byline={post.byline} quiet={bylineRepeats} />
        <App app={app} version={post.releaseVersion} />
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

type FiledApp = NonNullable<PostSummary["app"]> | null;

/** The app a filed line names, which is the one its byline has not already stated. */
function filedApp(post: PostSummary): FiledApp {
  if (!post.app) return null;
  if (post.releaseVersion) return post.app;
  return post.app.slug === post.byline.app?.slug ? null : post.app;
}

function App({ app, version }: { app: FiledApp; version?: string }) {
  if (!app) return null;
  return (
    <Link className={styles.app} href={`/blog/app/${app.slug}`}>
      {version ? `${app.name} ${version}` : app.name}
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
