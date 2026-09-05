import { ArticleContents } from "@/components/publication/ArticleContents";
import { ArticleIdentity } from "@/components/publication/ArticleIdentity";
import { PostBody } from "@/components/publication/PostBody";
import { ShareArticle } from "@/components/publication/ShareArticle";
import type { PublicPost } from "@/lib/api/query";
import { postContents } from "@/lib/post-contents";
import { asPostDocument } from "@/lib/post-document";
import { postPermalink } from "@/lib/post-link";
import styles from "./Article.module.css";
import { FurtherReading } from "./FurtherReading";
import { PublicationArt } from "./PublicationArt";

export function Article({ post }: { post: PublicPost }) {
  const document = asPostDocument(post.document);
  const contents = postContents(document);
  return (
    <article>
      <div className={styles.masthead}>
        {post.header ? null : <PublicationArt />}
        <div className={styles.column}>
          <ArticleIdentity
            byline={post.byline}
            category={post.category.label}
            header={post.header}
            media={post.media}
            publishedAt={post.publishedAt}
            release={post.release ?? null}
            summary={post.summary}
            title={post.title}
            updatedAt={post.updatedAt ?? null}
          />
        </div>
      </div>
      <div className={styles.column}>
        <div className={styles.reading}>
          <div className={styles.margin}>
            {contents.length > 0 ? (
              <ArticleContents entries={contents} />
            ) : null}
            <ShareArticle
              permalink={postPermalink(post.slug)}
              title={post.title}
            />
          </div>
          <div className={styles.prose}>
            <PostBody document={document} media={post.media} />
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
        <FurtherReading posts={post.related} />
      </div>
    </article>
  );
}
