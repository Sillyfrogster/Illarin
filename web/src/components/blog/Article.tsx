import { ArrowLeft, ArrowUp, Rss } from "lucide-react";
import Link from "next/link";
import { ArticleBody } from "@/components/blog/ArticleBody";
import { ArticleContents } from "@/components/blog/ArticleContents";
import { ArticleIdentity } from "@/components/blog/ArticleIdentity";
import { ShareArticle } from "@/components/blog/ShareArticle";
import { shellClasses } from "@/components/layout/Shell";
import type { PublicPost } from "@/lib/api/query";
import { postPermalink } from "@/lib/blog-address";
import { BLOG_FEEDS, BLOG_HOME } from "@/lib/blog-paths";
import { asPostBody } from "@/lib/post-body";
import { postContents } from "@/lib/post-contents";
import { FurtherReading } from "./FurtherReading";
import { PostPageActions } from "./PostPageActions";

export function Article({ post }: { post: PublicPost }) {
  const body = asPostBody(post.body);
  const contents = postContents(body);
  return (
    <article className={`${shellClasses} pb-16`} id="article-top">
      <div className="mt-6 flex items-center justify-between gap-4">
        <Link
          className="mt-6 inline-flex min-h-11 items-center gap-2 text-meta text-mute hover:text-ink"
          href={BLOG_HOME}
        >
          <ArrowLeft aria-hidden="true" className="size-4" />
          All posts
        </Link>
        <PostPageActions id={post.id} />
      </div>
      <div className="mx-auto max-w-[64rem] pt-6">
        <ArticleIdentity
          byline={post.byline}
          category={post.category.label}
          categorySlug={post.category.slug}
          header={post.header}
          media={post.media}
          publishedAt={post.publishedAt}
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
          <ArticleBody body={body} media={post.media} />
          <div className="mt-section flex flex-wrap items-center justify-between gap-4 rounded-plate bg-deep p-5">
            <p className="text-ui">Subscribe to blog updates.</p>
            <a
              className="flex min-h-11 items-center gap-2 text-ui font-medium text-accent hover:text-ink"
              href={BLOG_FEEDS.rss}
            >
              <Rss aria-hidden="true" className="size-4" />
              Subscribe via RSS
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
