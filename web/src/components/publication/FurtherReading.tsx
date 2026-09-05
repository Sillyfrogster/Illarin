import Link from "next/link";
import type { PostSummary } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./FurtherReading.module.css";

/** The few other posts an article points a reader at when they reach its end. */
export function FurtherReading({ posts }: { posts: PostSummary[] }) {
  if (posts.length === 0) return null;
  return (
    <section className={styles.further} aria-labelledby="further-reading">
      <h2 className={styles.heading} id="further-reading">
        Read next
      </h2>
      <ul className={styles.list}>
        {posts.map((post) => (
          <li className={styles.entry} key={post.id}>
            <h3 className={styles.title}>
              <Link className={styles.reach} href={`/blog/${post.slug}`}>
                {post.title}
              </Link>
            </h3>
            <p className={styles.filed}>
              <span className={styles.category}>{post.category.label}</span>
              {post.app ? (
                <span className={styles.app}>{post.app.name}</span>
              ) : null}
              <time dateTime={post.publishedAt}>
                {readableDate(post.publishedAt)}
              </time>
            </p>
          </li>
        ))}
      </ul>
    </section>
  );
}
