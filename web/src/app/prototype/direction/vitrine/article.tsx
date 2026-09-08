"use client";

import { ChevronRight } from "lucide-react";
import Image from "next/image";
import { Body } from "../body";
import { headings, useReadingPosition } from "../contents";
import { CATEGORY_LABEL, type Post, RICH_POSTS } from "../data";
import { EdgeBeam, RiseText } from "../motion";
import { SiteFoot, SiteHead } from "../shell";
import { Action, Byline, cn, Empty, Label, Rule } from "../ui";

function Contents({ post }: { post: Post }) {
  const list = headings(post.body);
  const active = useReadingPosition(list);
  if (list.length < 2) return null;
  return (
    <nav
      aria-label="Contents"
      className="vd:pointer-events-none vd:fixed vd:top-1/2 vd:left-6 vd:z-20 vd:hidden vd:-translate-y-1/2 vd:xl:block"
    >
      <ul className="vd:pointer-events-auto vd:flex vd:flex-col vd:gap-1 vd:pl-4 vd:shadow-[inset_1px_0_0_var(--v-rule)]">
        {list.map((entry) => (
          <li key={entry.id} className="vd:relative">
            <a
              href={`#${entry.id}`}
              aria-current={entry.id === active ? "true" : undefined}
              className={cn(
                "vd:block vd:max-w-44 vd:py-2 vd:text-meta vd:leading-5 vd:transition",
                entry.id === active
                  ? "vd:text-ink"
                  : "vd:text-faint vd:hover:text-mute",
              )}
            >
              {entry.text}
            </a>
            {entry.id === active && (
              <span
                aria-hidden="true"
                className="v-accent-rule vd:absolute vd:top-1/2 vd:-left-4 vd:h-px vd:w-3"
              />
            )}
          </li>
        ))}
      </ul>
    </nav>
  );
}

export function VitrineArticle({ post }: { post?: Post }) {
  if (!post) {
    return (
      <div id="top">
        <SiteHead direction="vitrine" section="Journal" />
        <div className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:md:px-10">
          <Empty title="No article" detail="Nothing to read here yet." />
        </div>
      </div>
    );
  }
  const next = RICH_POSTS.find((one) => one.id !== post.id);

  return (
    <div id="top">
      <SiteHead direction="vitrine" section="Journal" />
      <Contents post={post} />

      <article className="v-article vd:pt-14 vd:pb-chapter vd:md:pt-20">
        <header className="v-wide">
          <Label>{CATEGORY_LABEL[post.category]}</Label>
          <h1 className="vd:mt-6 vd:max-w-[20ch] vd:font-display vd:text-hero vd:font-medium">
            <RiseText text={post.title} />
          </h1>
          <p className="v-lede vd:mt-7 vd:font-prose vd:text-lede vd:text-mute">
            {post.dek}
          </p>
          <div className="vd:mt-8 vd:flex vd:flex-wrap vd:items-center vd:justify-between vd:gap-5">
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
          <div className="vd:mt-10">
            <Rule accent />
          </div>
        </header>

        {post.image && (
          <figure className="v-full vd:mx-auto vd:mt-10 vd:w-full vd:max-w-[76rem]">
            <span className="vd:relative vd:block vd:overflow-hidden vd:rounded-plate">
              <Image
                src={post.image}
                alt=""
                priority
                sizes="(max-width: 76rem) 100vw, 76rem"
                className="vd:h-auto vd:w-full"
              />
              <EdgeBeam seconds={12} />
            </span>
          </figure>
        )}

        <div className="v-wide vd:h-section" />

        {post.body ? (
          <Body blocks={post.body} direction="vitrine" />
        ) : (
          <Empty
            title="This one is still being written"
            detail="The summary above is all there is for now."
          />
        )}
      </article>

      {next && (
        <section className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:md:px-10">
          <Rule />
          <a
            href="#top"
            className="v-row v-row-shift vd:group vd:grid vd:gap-6 vd:py-10 vd:lg:grid-cols-12 vd:lg:items-center"
          >
            <div className="vd:lg:col-span-7">
              <Label>Next in the journal</Label>
              <h2 className="vd:mt-4 vd:max-w-[18ch] vd:font-display vd:text-display vd:font-medium">
                {next.title}
              </h2>
              <p className="vd:mt-4 vd:max-w-[56ch] vd:text-[1rem] vd:leading-7 vd:text-mute">
                {next.dek}
              </p>
              <span className="vd:mt-6 vd:inline-flex vd:items-center vd:gap-1.5 vd:text-ui vd:font-semibold vd:text-ink">
                Read it
                <ChevronRight className="vd:size-4 vd:transition vd:group-hover:translate-x-0.5" />
              </span>
            </div>
            {next.image && (
              <div className="vd:overflow-hidden vd:rounded-plate vd:lg:col-span-4 vd:lg:col-start-9">
                <Image
                  src={next.image}
                  alt=""
                  sizes="(max-width: 64rem) 100vw, 28rem"
                  className="vd:h-auto vd:w-full"
                />
              </div>
            )}
          </a>
        </section>
      )}

      <SiteFoot />
    </div>
  );
}
