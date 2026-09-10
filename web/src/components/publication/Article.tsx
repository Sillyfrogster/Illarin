import { ArrowLeft, ArrowUp, Rss } from "lucide-react";
import Link from "next/link";
import { shellClasses } from "@/components/layout/Shell";
import { ArticleContents } from "@/components/publication/ArticleContents";
import { ArticleIdentity } from "@/components/publication/ArticleIdentity";
import { PostBody } from "@/components/publication/PostBody";
import { ShareArticle } from "@/components/publication/ShareArticle";
import type { PublicPost } from "@/lib/api/query";
import { postContents } from "@/lib/post-contents";
import { asPostDocument } from "@/lib/post-document";
import { postPermalink } from "@/lib/post-link";
import { BLOG_HOME, PUBLICATION_FEEDS } from "@/lib/publication-metadata";
import { FurtherReading } from "./FurtherReading";

export function Article({ post }: { post: PublicPost }) {
  const document = asPostDocument(post.document);
  const contents = postContents(document);
  return (
    <article className={`${shellClasses} pb-16`} id="article-top">
      <Link
        className="mt-6 inline-flex min-h-11 items-center gap-2 text-meta text-mute hover:text-ink"
        href={BLOG_HOME}
      >
        <ArrowLeft aria-hidden="true" className="size-4" />
        All posts
      </Link>
      <div className="mx-auto max-w-[64rem] pt-6">
        <ArticleIdentity
          byline={post.byline}
          category={post.category.label}
          categorySlug={post.category.slug}
          header={post.header}
          media={post.media}
          publishedAt={post.publishedAt}
          release={post.release ?? null}
          summary={post.summary}
          title={post.title}
          updatedAt={post.updatedAt ?? null}
        />
      </div>
      <div className="mx-auto mt-group grid max-w-[64rem] items-start gap-x-12 gap-y-6 lg:grid-cols-[13rem_minmax(0,1fr)]">
        <aside className="grid gap-4 lg:sticky lg:top-[calc(var(--header-height)+2rem)]">
          {contents.length > 0 ? <ArticleContents entries={contents} /> : null}
          <ShareArticle
            permalink={postPermalink(post.slug)}
            title={post.title}
          />
        </aside>
        <div className="min-w-0 max-w-[70ch]">
          <PostBody document={document} media={post.media} />
          {post.release?.address ? (
            <p className="mt-group text-ui">
              <a
                className="text-ink underline decoration-accent/55 underline-offset-[3px] hover:decoration-accent"
                href={post.release.address}
                rel="noreferrer noopener"
                target="_blank"
              >
                {post.release.app.name} {post.release.version} release notes
              </a>
            </p>
          ) : null}
          <div className="mt-section flex flex-wrap items-center justify-between gap-4 rounded-plate bg-deep p-5">
            <p className="text-ui">Keep up with Illarin.</p>
            <a
              className="flex min-h-11 items-center gap-2 text-ui font-medium text-accent hover:text-ink"
              href={PUBLICATION_FEEDS.rss}
            >
              <Rss aria-hidden="true" className="size-4" />
              Follow the feed
            </a>
          </div>
          <FurtherReading posts={post.related} />
          <a
            className="mt-group inline-flex min-h-11 items-center gap-2 text-meta text-mute hover:text-ink"
            href="#article-top"
          >
            Back to top
            <ArrowUp aria-hidden="true" className="size-4" />
          </a>
        </div>
      </div>
    </article>
  );
}
