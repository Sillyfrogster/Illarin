"use client";

import { ChevronDown } from "lucide-react";
import Image from "next/image";
import { Body } from "../body";
import { headings } from "../contents";
import { CATEGORY_LABEL, type Post, RICH_POSTS } from "../data";
import { Room } from "../motion";
import { SiteFoot, SiteHead } from "../shell";
import { Action, Byline, cn, Empty, Label, tintStyle } from "../ui";

function Contents({ post }: { post: Post }) {
  const list = headings(post.body);
  if (list.length < 2) return null;
  return (
    <details className="vd:group vd:mt-2">
      <summary className="vd:inline-flex vd:min-h-11 vd:cursor-pointer vd:list-none vd:items-center vd:gap-2 vd:rounded-control vd:px-4 vd:text-ui vd:font-semibold vd:shadow-[inset_0_0_0_1px_var(--v-rule)]">
        Contents
        <span className="v-tabular vd:text-mute">{list.length}</span>
        <ChevronDown className="vd:size-4 vd:transition vd:group-open:rotate-180 vd:motion-reduce:transition-none" />
      </summary>
      <ol className="vd:mt-4 vd:grid vd:gap-x-8 vd:gap-y-1 vd:sm:grid-cols-2">
        {list.map((entry) => (
          <li key={entry.id}>
            <a
              href={`#${entry.id}`}
              className="vd:flex vd:min-h-10 vd:items-center vd:gap-3 vd:text-ui vd:text-mute vd:hover:text-ink"
            >
              <span
                className="v-tabular vd:text-meta"
                style={{ color: "var(--v-tint-ink)" }}
              >
                {String(entry.number).padStart(2, "0")}
              </span>
              {entry.text}
            </a>
          </li>
        ))}
      </ol>
    </details>
  );
}

export function AmbientArticle({
  post,
  theme,
}: {
  post?: Post;
  theme: "light" | "dark";
}) {
  if (!post) {
    return (
      <div id="top">
        <SiteHead direction="ambient" section="Journal" />
        <div className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:md:px-10">
          <Empty title="No article" detail="Nothing to read here yet." />
        </div>
      </div>
    );
  }
  const dark = theme === "dark";
  const more = RICH_POSTS.filter((one) => one.id !== post.id).slice(0, 3);

  return (
    <div id="top" style={tintStyle(post.tint, dark)}>
      <SiteHead
        direction="ambient"
        section="Journal"
        over={Boolean(post.image)}
      />

      <article>
        {post.image ? (
          <header className="vd:relative vd:-mt-[4.5rem] vd:flex vd:min-h-[68dvh] vd:items-end vd:overflow-hidden">
            <Image
              src={post.image}
              alt=""
              priority
              sizes="100vw"
              className="vd:absolute vd:inset-0 vd:size-full vd:object-cover"
            />
            <div className="v-scrim vd:absolute vd:inset-0" />
            <div
              aria-hidden="true"
              className="vd:absolute vd:inset-x-0 vd:bottom-0 vd:h-40 vd:bg-linear-to-t vd:from-field vd:to-transparent"
            />
            <div className="vd:relative vd:mx-auto vd:w-full vd:max-w-[86rem] vd:px-5 vd:pt-24 vd:pb-12 vd:md:px-10 vd:md:pb-16">
              <Label className="vd:text-over-mute">
                {CATEGORY_LABEL[post.category]}
              </Label>
              <h1 className="vd:mt-5 vd:max-w-[20ch] vd:font-display vd:text-hero vd:font-medium vd:text-over">
                {post.title}
              </h1>
              <p className="v-lede vd:mt-6 vd:font-prose vd:text-lede vd:text-over-mute">
                {post.dek}
              </p>
            </div>
          </header>
        ) : (
          <Room className="vd:px-5 vd:pt-16 vd:md:px-10">
            <div className="vd:mx-auto vd:max-w-[86rem]">
              <Label style={{ color: "var(--v-tint-ink)" }}>
                {CATEGORY_LABEL[post.category]}
              </Label>
              <h1 className="vd:mt-5 vd:max-w-[20ch] vd:font-display vd:text-hero vd:font-medium">
                {post.title}
              </h1>
              <p className="v-lede vd:mt-6 vd:font-prose vd:text-lede vd:text-mute">
                {post.dek}
              </p>
            </div>
          </Room>
        )}

        <Room>
          <div className="v-article vd:pt-8">
            <div className="v-wide vd:flex vd:flex-wrap vd:items-center vd:justify-between vd:gap-x-8 vd:gap-y-4">
              <Byline
                author={post.author}
                handle={post.handle}
                category={post.category}
                date={post.date}
                updated={post.updated}
                minutes={post.minutes}
              />
              <Action variant="secondary">Copy link</Action>
            </div>
            <div className="v-wide vd:mt-6">
              <Contents post={post} />
            </div>
            <div className="v-wide vd:h-group" />
            {post.body ? (
              <Body blocks={post.body} direction="ambient" />
            ) : (
              <Empty
                title="This one is still being written"
                detail="The summary above is all there is for now."
              />
            )}
            <div className="v-wide vd:h-chapter" />
          </div>

          {Boolean(more.length) && (
            <section className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:md:px-10">
              <h2 className="vd:font-display vd:text-display vd:font-medium">
                Keep reading
              </h2>
              <div className="vd:mt-group vd:grid vd:gap-5 vd:sm:grid-cols-3">
                {more.map((one) => (
                  <a
                    key={one.id}
                    href="#top"
                    style={tintStyle(one.tint, dark)}
                    className="vd:group vd:block"
                  >
                    <div
                      className={cn(
                        "vd:relative vd:aspect-16/10 vd:overflow-hidden vd:rounded-plate",
                        !one.image && "vd:bg-accent-wash",
                      )}
                    >
                      {one.image && (
                        <Image
                          src={one.image}
                          alt=""
                          sizes="(max-width: 40rem) 100vw, 24rem"
                          className="vd:size-full vd:object-cover vd:transition vd:duration-700 vd:group-hover:scale-[1.03] vd:motion-reduce:transition-none"
                        />
                      )}
                    </div>
                    <Label
                      className="vd:mt-4"
                      style={{ color: "var(--v-tint-ink)" }}
                    >
                      {CATEGORY_LABEL[one.category]}
                    </Label>
                    <h3 className="vd:mt-2 vd:font-display vd:text-[1.375rem] vd:leading-[1.2] vd:font-medium vd:wrap-pretty">
                      {one.title}
                    </h3>
                    <p className="vd:mt-2 vd:line-clamp-2 vd:text-meta vd:leading-6 vd:text-mute">
                      {one.dek}
                    </p>
                  </a>
                ))}
              </div>
            </section>
          )}

          <SiteFoot />
        </Room>
      </article>
    </div>
  );
}
