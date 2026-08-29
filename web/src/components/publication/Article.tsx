import { ArticleIdentity } from "@/components/publication/ArticleIdentity";
import { PostBody } from "@/components/publication/PostBody";
import type { PublicPost } from "@/lib/api/query";
import { asPostDocument } from "@/lib/post-document";
import styles from "./Article.module.css";

export function Article({ post }: { post: PublicPost }) {
  return (
    <article>
      <div className={styles.masthead}>
        <span aria-hidden="true" className={styles.art} />
        <div className={styles.column}>
          <ArticleIdentity
            byline={post.byline}
            category={post.category.label}
            publishedAt={post.publishedAt}
            release={post.release ?? null}
            summary={post.summary}
            title={post.title}
            updatedAt={post.updatedAt ?? null}
          />
        </div>
      </div>
      <div className={styles.column}>
        <div className={styles.prose}>
          <PostBody document={asPostDocument(post.document)} />
          {post.release?.address ? (
            <p className={styles.releaseLink}>
              <a
                href={post.release.address}
                rel="noreferrer noopener"
                target="_blank"
              >
                {post.release.app.name} {post.release.version} release notes
              </a>
            </p>
          ) : null}
        </div>
      </div>
    </article>
  );
}
